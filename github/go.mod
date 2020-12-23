module github.com/GoogleCloudPlatform/cloud-build-notifiers/github

go 1.15

require (
	cloud.google.com/go v0.73.0
	cloud.google.com/go/storage v1.12.0
	github.com/GoogleCloudPlatform/cloud-build-notifiers/lib/notifiers v0.0.0-20201207173907-e18059bc9a58
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.3
	github.com/google/go-github/v32 v32.1.0
	github.com/tektoncd/pipeline v0.19.0
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/test-infra v0.0.0-20200803112140-d8aa4e063646
	knative.dev/pkg v0.0.0-20201223002104-9d0775512af8
)
