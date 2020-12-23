// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-build-notifiers/lib/notifiers"
	"github.com/bradleyfalzon/ghinstallation"
	log "github.com/golang/glog"
	"github.com/google/go-github/v32/github"
	pb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

func main() {
	if err := notifiers.Main(new(notifier)); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

type notifier struct {
	filter notifiers.EventFilter
	atr    *ghinstallation.AppsTransport
}

func (n *notifier) SetUp(ctx context.Context, cfg *notifiers.Config, secrets notifiers.SecretGetter, _ notifiers.BindingResolver) error {
	prd, err := notifiers.MakeCELPredicate(cfg.Spec.Notification.Filter)
	if err != nil {
		return fmt.Errorf("failed to create CELPredicate: %w", err)
	}
	n.filter = prd

	// Create new installation transport.
	raw, ok := cfg.Spec.Notification.Delivery["app_id"].(string)
	if !ok {
		return fmt.Errorf("expected delivery config %v to have string field `app_id`", cfg.Spec.Notification.Delivery)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return err
	}

	key, err := secrets.GetSecret(ctx, cfg.Spec.Notification.Delivery["app_key"].(string))
	if err != nil {
		return err
	}

	tr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, id, []byte(key))
	if err != nil {
		return err
	}
	n.atr = tr

	return nil
}

func (n *notifier) SendNotification(ctx context.Context, b *pb.Build) error {
	if !n.filter.Apply(ctx, b) {
		log.V(2).Infof("not sending GitHub request for event (build id = %s, status = %v)", b.GetId(), b.GetStatus())
		return nil
	}

	log.Infof("sending GitHub request for event (build id = %s, status = %s)", b.GetId(), b.GetStatus())

	t, err := trigger(ctx, b)
	if err != nil {
		return err
	}
	client, err := n.githubClient(ctx, b, t)
	if err != nil {
		return err
	}

	body, err := renderTemplate(b)
	if err != nil {
		return err
	}

	output := &github.CheckRunOutput{
		Title:   github.String(t.GetName()),
		Summary: github.String(body),
	}
	if logs, err := logs(ctx, b); err == nil {
		// We can't guarantee we have the ability to read logs, so add them best effort.
		output.Text = github.String(fmt.Sprintf("```\n%s\n```", logs))
	}

	_, err = client.UpsertCheckRun(ctx, b, t, output)
	return err
}

func trigger(ctx context.Context, b *pb.Build) (*pb.BuildTrigger, error) {
	fmt.Printf("%+v\n", b)

	// Cloud Build does not provide event information in the Build, so we have to infer it from the trigger.
	trigger := b.GetBuildTriggerId()
	if trigger == "" {
		return nil, errors.New("build is missing trigger, cannot infer source information")
	}
	gcb, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return gcb.GetBuildTrigger(ctx, &pb.GetBuildTriggerRequest{
		ProjectId: b.GetProjectId(),
		TriggerId: b.GetBuildTriggerId(),
	})
}

func logs(ctx context.Context, b *pb.Build) (string, error) {
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	r, err := gcs.Bucket(strings.TrimPrefix(b.GetLogsBucket(), "gs://")).Object(fmt.Sprintf("log-%s.txt", b.GetId())).NewReader(ctx)
	if err != nil {
		return "", err
	}
	bLog, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(bLog), nil
}
