package version

// force tools into vendor dir
import (
	_ "github.com/golang/mock/mockgen"
	_ "go.medium.engineering/kubernetes/cmd/matcher-gen"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)

var (
	Version = "0.0.1"
)
