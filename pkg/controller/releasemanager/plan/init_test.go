package plan

import (
	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	if err := istiov1alpha3.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := monitoringv1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := slov1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}
