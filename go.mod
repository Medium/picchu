module go.medium.engineering/picchu

go 1.14

require (
	github.com/Medium/service-level-operator v0.3.1-0.20200128160720-77476ad50a61
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/coreos/prometheus-operator v0.38.1
	github.com/go-logr/logr v0.2.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.7
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.3
	github.com/google/uuid v1.1.1
	github.com/operator-framework/operator-sdk v0.13.0
	github.com/practo/k8s-worker-pod-autoscaler v1.0.1-0.20200714064616-7a81cf1a4cb8
	github.com/prometheus/client_golang v1.7.0
	github.com/prometheus/common v0.10.0
	github.com/prometheus/prometheus v2.3.2+incompatible
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.0
	go.medium.engineering/kubernetes v0.0.0-20200708143024-6f9f0aae51b2
	go.uber.org/zap v1.13.0
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	gomodules.xyz/jsonpatch/v2 v2.0.1
	istio.io/api v0.0.0-20200617184712-fb83ff2d8228
	istio.io/client-go v0.0.0-20200615164228-d77b0b53b6a0
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.3
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451
	sigs.k8s.io/controller-runtime v0.6.0
)

// Pinned to kubernetes-1.17
replace (
	github.com/golang/mock => github.com/golang/mock v1.3.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200701144905-de5b010b2b38
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20191111142012-edeb7a44cbf7
	k8s.io/api => k8s.io/api v0.17.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.8
	k8s.io/apiserver => k8s.io/apiserver v0.17.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.8
	k8s.io/client-go => k8s.io/client-go v0.17.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.8
	k8s.io/code-generator => k8s.io/code-generator v0.17.8
	k8s.io/component-base => k8s.io/component-base v0.17.8
	k8s.io/cri-api => k8s.io/cri-api v0.17.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.8
	k8s.io/kubectl => k8s.io/kubectl v0.17.8
	k8s.io/kubelet => k8s.io/kubelet v0.17.8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.8
	k8s.io/metrics => k8s.io/metrics v0.17.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.8
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.7
)
