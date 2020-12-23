package main

import (
	"context"
	"fmt"
	"testing"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/GoogleCloudPlatform/cloud-build-notifiers/lib/notifiers"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	pb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

type secretClient struct {
	*secretmanager.Client
}

func (a *secretClient) GetSecret(ctx context.Context, name string) (string, error) {
	fmt.Println(name)
	res, err := a.AccessSecretVersion(ctx, &smpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("failed to get secret named %q: %w", name, err)
	}

	return string(res.GetPayload().GetData()), nil
}

func TestGitHub(t *testing.T) {
	ctx := context.Background()
	cfg := &notifiers.Config{
		Spec: &notifiers.Spec{
			Notification: &notifiers.Notification{
				Filter: "true",
				Delivery: map[string]interface{}{
					"app_id":  "9994",
					"app_key": "projects/158923220293/secrets/github-app-wlynch/versions/1",
				},
			},
		},
	}

	n := notifier{}
	smc, err := secretmanager.NewClient(ctx)
	if err != nil {
		t.Fatalf("failed to create new SecretManager client: %v", err)
	}

	if err := n.SetUp(context.Background(), cfg, &secretClient{Client: smc}, nil); err != nil {
		t.Fatal(err)
	}

	gcb, _ := cloudbuild.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := gcb.GetBuild(ctx, &pb.GetBuildRequest{
		ProjectId: "wlynch-test",
		Id:        "9208dca6-36a6-4268-9c63-6f3e73a5efcf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := n.SendNotification(ctx, b); err != nil {
		t.Error(err)
	}
}
