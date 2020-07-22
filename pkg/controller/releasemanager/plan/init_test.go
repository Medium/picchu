package plan

import (
	"time"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	coreAsserts "go.medium.engineering/kubernetes/pkg/test/core/v1"
	istioAsserts "go.medium.engineering/kubernetes/pkg/test/istio/networking/v1alpha3"
	"go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	picchuScheme "go.medium.engineering/picchu/pkg/client/scheme"
	monitoringAsserts "go.medium.engineering/picchu/pkg/test/monitoring/v1"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	scalingFactor = 1.0
	cluster       = &picchu.Cluster{
		Spec: picchu.ClusterSpec{
			ScalingFactor: &scalingFactor,
		},
	}
	halfScalingFactor = 0.5
	halfCluster       = &picchu.Cluster{
		Spec: picchu.ClusterSpec{
			ScalingFactor: &halfScalingFactor,
		},
	}
	scheme     = runtime.NewScheme()
	timeout    = time.Duration(10000000) * time.Second
	comparator = ktest.NewComparator(scheme)
)

func init() {
	// TODO: Deep in the logging code, we use the picchuScheme, so it must be registered too until that's fixed.
	for _, s := range []*runtime.Scheme{scheme, picchuScheme.Scheme} {
		if err := core.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := apps.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := autoscaling.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := picchu.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := istiov1alpha3.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := monitoringv1.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := v1alpha1.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := slov1alpha1.AddToScheme(s); err != nil {
			panic(err)
		}
		if err := wpav1.AddToScheme(s); err != nil {
			panic(err)
		}
	}
	monitoringAsserts.RegisterAsserts(comparator)
	coreAsserts.RegisterAsserts(comparator)
	istioAsserts.RegisterAsserts(comparator)
}

func fakeClient(objs ...runtime.Object) client.Client {
	return fake.NewFakeClientWithScheme(scheme, objs...)
}
