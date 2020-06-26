package v1

import (
	testing "testing"

	v1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	assert "github.com/stretchr/testify/assert"
	test "go.medium.engineering/kubernetes/pkg/test"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	if err := v1.AddToScheme(test.DefaultScheme); err != nil {
		panic(err)
	}
	RegisterAsserts(test.DefaultComparator)
}

func RegisterAsserts(comparator *test.Comparator) {
	comparator.RegisterForType(&v1.Alertmanager{}, test.TypedAsserts{
		Match: func(t *testing.T, a, b runtime.Object) {
			Match_Alertmanager(t, a.(*v1.Alertmanager), b.(*v1.Alertmanager))
		},
		NoMatch: func(t *testing.T, a, b runtime.Object) {
			NoMatch_Alertmanager(t, a.(*v1.Alertmanager), b.(*v1.Alertmanager))
		},
	})

	comparator.RegisterForType(&v1.PodMonitor{}, test.TypedAsserts{
		Match: func(t *testing.T, a, b runtime.Object) {
			Match_PodMonitor(t, a.(*v1.PodMonitor), b.(*v1.PodMonitor))
		},
		NoMatch: func(t *testing.T, a, b runtime.Object) {
			NoMatch_PodMonitor(t, a.(*v1.PodMonitor), b.(*v1.PodMonitor))
		},
	})

	comparator.RegisterForType(&v1.Prometheus{}, test.TypedAsserts{
		Match: func(t *testing.T, a, b runtime.Object) {
			Match_Prometheus(t, a.(*v1.Prometheus), b.(*v1.Prometheus))
		},
		NoMatch: func(t *testing.T, a, b runtime.Object) {
			NoMatch_Prometheus(t, a.(*v1.Prometheus), b.(*v1.Prometheus))
		},
	})

	comparator.RegisterForType(&v1.ThanosRuler{}, test.TypedAsserts{
		Match: func(t *testing.T, a, b runtime.Object) {
			Match_ThanosRuler(t, a.(*v1.ThanosRuler), b.(*v1.ThanosRuler))
		},
		NoMatch: func(t *testing.T, a, b runtime.Object) {
			NoMatch_ThanosRuler(t, a.(*v1.ThanosRuler), b.(*v1.ThanosRuler))
		},
	})

	comparator.RegisterForType(&v1.ServiceMonitor{}, test.TypedAsserts{
		Match: func(t *testing.T, a, b runtime.Object) {
			Match_ServiceMonitor(t, a.(*v1.ServiceMonitor), b.(*v1.ServiceMonitor))
		},
		NoMatch: func(t *testing.T, a, b runtime.Object) {
			NoMatch_ServiceMonitor(t, a.(*v1.ServiceMonitor), b.(*v1.ServiceMonitor))
		},
	})

	comparator.RegisterForType(&v1.PrometheusRule{}, test.TypedAsserts{
		Match: func(t *testing.T, a, b runtime.Object) {
			Match_PrometheusRule(t, a.(*v1.PrometheusRule), b.(*v1.PrometheusRule))
		},
		NoMatch: func(t *testing.T, a, b runtime.Object) {
			NoMatch_PrometheusRule(t, a.(*v1.PrometheusRule), b.(*v1.PrometheusRule))
		},
	})

}

func Assimilate_Alertmanager(expected, actual *v1.Alertmanager) *v1.Alertmanager {
	e := expected.DeepCopyObject().(*v1.Alertmanager)
	e.ObjectMeta = test.Assimilate_ObjectMeta(e.ObjectMeta, actual.ObjectMeta)
	return e
}

func Match_Alertmanager(t *testing.T, expected, actual *v1.Alertmanager) {
	assert := assert.New(t)
	e := Assimilate_Alertmanager(expected, actual)
	assert.EqualValues(e, actual)
}

func NoMatch_Alertmanager(t *testing.T, expected, actual *v1.Alertmanager) {
	assert := assert.New(t)
	e := Assimilate_Alertmanager(expected, actual)
	assert.NotEqualValues(e, actual)
}

func Assimilate_PodMonitor(expected, actual *v1.PodMonitor) *v1.PodMonitor {
	e := expected.DeepCopyObject().(*v1.PodMonitor)
	e.ObjectMeta = test.Assimilate_ObjectMeta(e.ObjectMeta, actual.ObjectMeta)
	return e
}

func Match_PodMonitor(t *testing.T, expected, actual *v1.PodMonitor) {
	assert := assert.New(t)
	e := Assimilate_PodMonitor(expected, actual)
	assert.EqualValues(e, actual)
}

func NoMatch_PodMonitor(t *testing.T, expected, actual *v1.PodMonitor) {
	assert := assert.New(t)
	e := Assimilate_PodMonitor(expected, actual)
	assert.NotEqualValues(e, actual)
}

func Assimilate_Prometheus(expected, actual *v1.Prometheus) *v1.Prometheus {
	e := expected.DeepCopyObject().(*v1.Prometheus)
	e.ObjectMeta = test.Assimilate_ObjectMeta(e.ObjectMeta, actual.ObjectMeta)
	return e
}

func Match_Prometheus(t *testing.T, expected, actual *v1.Prometheus) {
	assert := assert.New(t)
	e := Assimilate_Prometheus(expected, actual)
	assert.EqualValues(e, actual)
}

func NoMatch_Prometheus(t *testing.T, expected, actual *v1.Prometheus) {
	assert := assert.New(t)
	e := Assimilate_Prometheus(expected, actual)
	assert.NotEqualValues(e, actual)
}

func Assimilate_ThanosRuler(expected, actual *v1.ThanosRuler) *v1.ThanosRuler {
	e := expected.DeepCopyObject().(*v1.ThanosRuler)
	e.ObjectMeta = test.Assimilate_ObjectMeta(e.ObjectMeta, actual.ObjectMeta)
	return e
}

func Match_ThanosRuler(t *testing.T, expected, actual *v1.ThanosRuler) {
	assert := assert.New(t)
	e := Assimilate_ThanosRuler(expected, actual)
	assert.EqualValues(e, actual)
}

func NoMatch_ThanosRuler(t *testing.T, expected, actual *v1.ThanosRuler) {
	assert := assert.New(t)
	e := Assimilate_ThanosRuler(expected, actual)
	assert.NotEqualValues(e, actual)
}

func Assimilate_ServiceMonitor(expected, actual *v1.ServiceMonitor) *v1.ServiceMonitor {
	e := expected.DeepCopyObject().(*v1.ServiceMonitor)
	e.ObjectMeta = test.Assimilate_ObjectMeta(e.ObjectMeta, actual.ObjectMeta)
	return e
}

func Match_ServiceMonitor(t *testing.T, expected, actual *v1.ServiceMonitor) {
	assert := assert.New(t)
	e := Assimilate_ServiceMonitor(expected, actual)
	assert.EqualValues(e, actual)
}

func NoMatch_ServiceMonitor(t *testing.T, expected, actual *v1.ServiceMonitor) {
	assert := assert.New(t)
	e := Assimilate_ServiceMonitor(expected, actual)
	assert.NotEqualValues(e, actual)
}

func Assimilate_PrometheusRule(expected, actual *v1.PrometheusRule) *v1.PrometheusRule {
	e := expected.DeepCopyObject().(*v1.PrometheusRule)
	e.ObjectMeta = test.Assimilate_ObjectMeta(e.ObjectMeta, actual.ObjectMeta)
	return e
}

func Match_PrometheusRule(t *testing.T, expected, actual *v1.PrometheusRule) {
	assert := assert.New(t)
	e := Assimilate_PrometheusRule(expected, actual)
	assert.EqualValues(e, actual)
}

func NoMatch_PrometheusRule(t *testing.T, expected, actual *v1.PrometheusRule) {
	assert := assert.New(t)
	e := Assimilate_PrometheusRule(expected, actual)
	assert.NotEqualValues(e, actual)
}
