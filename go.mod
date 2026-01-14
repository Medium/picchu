module go.medium.engineering/picchu

go 1.25.5

require (
	github.com/DataDog/datadog-api-client-go/v2 v2.42.0
	github.com/DataDog/datadog-operator/api v0.0.0-20251110224013-76f698af3867
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/external-secrets/external-secrets v0.16.2
	github.com/go-logr/logr v1.4.3
	github.com/go-logr/zapr v1.3.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.6.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.38.2
	github.com/practo/k8s-worker-pod-autoscaler v1.6.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.86.1
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.67.2
	github.com/prometheus/prometheus v0.307.2
	github.com/slack-go/slack v0.17.3
	github.com/slok/sloth v0.15.0
	github.com/stretchr/testify v1.11.1
	go.medium.engineering/kubernetes v0.1.9
	go.uber.org/mock v0.6.0
	go.uber.org/zap v1.27.1
	golang.org/x/sync v0.18.0
	istio.io/api v0.0.0-20220322234440-289bfe748e00
	istio.io/client-go v1.13.2
	k8s.io/api v0.34.2
	k8s.io/apimachinery v0.34.2
	k8s.io/client-go v0.34.2
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4
	sigs.k8s.io/controller-runtime v0.22.3
)

require github.com/kedacore/keda/v2 v2.18.3

require (
	github.com/DataDog/zstd v1.5.6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/expr-lang/expr v1.17.6 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/jsonreference v0.21.2 // indirect
	github.com/go-openapi/swag v0.25.1 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.1 // indirect
	github.com/go-openapi/swag/conv v0.25.1 // indirect
	github.com/go-openapi/swag/fileutils v0.25.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.1 // indirect
	github.com/go-openapi/swag/loading v0.25.1 // indirect
	github.com/go-openapi/swag/mangling v0.25.1 // indirect
	github.com/go-openapi/swag/netutils v0.25.1 // indirect
	github.com/go-openapi/swag/stringutils v0.25.1 // indirect
	github.com/go-openapi/swag/typeutils v0.25.1 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/oauth2 v0.33.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	istio.io/gogo-genproto v0.0.0-20211208193508-5ab4acc9eb1e // indirect
	k8s.io/apiextensions-apiserver v0.34.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	knative.dev/pkg v0.0.0-20250326102644-9f3e60a9244c // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.14.0

// replace k8s.io/api => k8s.io/api v0.29.6

// replace k8s.io/apimachinery => k8s.io/apimachinery v0.29.6

// replace k8s.io/client-go => k8s.io/client-go v0.29.6

// replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.29.6

// replace k8s.io/component-base => k8s.io/component-base v0.26.0

// replace sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.20.4
