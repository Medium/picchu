module go.medium.engineering/picchu

go 1.14

require (
	github.com/Medium/service-level-operator v0.3.1-0.20200128160720-77476ad50a61
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/coreos/prometheus-operator v0.40.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.3
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.3
	github.com/google/uuid v1.1.1
	github.com/operator-framework/operator-sdk v0.18.1
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.9.1
	github.com/prometheus/prometheus v2.3.2+incompatible
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.0
	go.medium.engineering/kubernetes v0.0.0-20200625160729-4b40d46f5d68
	go.uber.org/zap v1.15.0
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	istio.io/api v0.0.0-20200617184712-fb83ff2d8228
	istio.io/client-go v0.0.0-20200615164228-d77b0b53b6a0
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.2
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20200213233353-b90be6f32a33
	k8s.io/client-go => k8s.io/client-go v0.18.2
)
