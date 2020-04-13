package plan

import (
	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	scalingFactor = 1.0
	cluster       = &picchuv1alpha1.Cluster{
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &scalingFactor,
		},
	}
	halfScalingFactor = 0.5
	halfCluster       = &picchuv1alpha1.Cluster{
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &halfScalingFactor,
		},
	}
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

func fakeClient(objs ...runtime.Object) client.Client {
	return fake.NewFakeClientWithScheme(scheme.Scheme, objs...)
}
