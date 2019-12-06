module go.medium.engineering/picchu

go 1.13

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/aws/aws-sdk-go-v2 v0.17.0
	github.com/coreos/prometheus-operator v0.29.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.0
	github.com/golang/mock v1.3.1
	github.com/google/uuid v1.0.0
	github.com/knative/pkg v0.0.0-20190308001241-2b411285d2b9 // outdated
	github.com/operator-framework/operator-sdk v0.12.1-0.20191112211508-82fc57de5e5b
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.4.1
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0
	go.uber.org/zap v1.10.0
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	sigs.k8s.io/controller-runtime v0.3.0
)

// Pinned to kubernetes-1.15.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918200908-1e17798da8c1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918202139-0b14c719ca62
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918203125-ae665f80358a
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918202959-c340507a5d48
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190918200425-ed2f0867c778
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918203248-97c07dcbb623
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918201136-c3a845f1fbb2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918202837-c54ce30c680e
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918202429-08c8357f8e2d
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918202713-c34a54b3ec8e
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918202550-958285cf3eef
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918203421-225f0541b3ea
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190918202012-3c1ca76f5bda
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918201353-5cc279503896
)

// // Pinned to kubernetes-1.14
// //
// // find "Pseudo-versions" using `go list -m k8s.io/<package_name>@<git_tag>`
// // example: go list -u k8s.io/api@release-1.14
// replace (
// 	k8s.io/api => k8s.io/api v0.0.0-20191004102349-159aefb8556b
// 	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191004105649-b14e3c49469a
// 	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689
// 	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191109015554-8577c320c87f
// 	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191004110135-b9eb767d2e1a
// 	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a // tag release-14.0
// 	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191004111010-9775d7be8494
// 	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190816225014-88e17f53ad9d
// 	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190704094409-6c2a4329ac29
// 	k8s.io/component-base => k8s.io/component-base v0.0.0-20190816222507-f3799749b6b7
// 	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79 // tag release-1.15
// 	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190816225257-04c685fc1cd2
// 	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191107015716-808b5b5e73bb
// 	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190816224923-d5aa5a9bfcd3
// 	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190816224646-61013d27312f
// 	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190816224829-b006ac710708
// 	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190816224737-a8b37c94716b
// 	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191205122231-0fccbcaf5ea7 // tag release-1.15
// 	k8s.io/metrics => k8s.io/metrics v0.0.0-20191004105854-2e8cf7d0888c
// 	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191004104527-8ecaca529818
// )
