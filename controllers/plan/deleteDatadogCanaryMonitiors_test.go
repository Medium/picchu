package plan

import (
	"context"
	_ "runtime"
	"testing"

	ktest "go.medium.engineering/kubernetes/pkg/test"

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteDatadogCanaryMonitors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteDatadogCanaryMonitor := &DeleteDatadogCanaryMonitors{
		App:       "echo",
		Namespace: "datadog",
	}
	sl := &ddog.DatadogMonitor{
		ObjectMeta: meta.ObjectMeta{
			Name:      "slo1",
			Namespace: "datadog",
			Labels: map[string]string{
				picchu.LabelApp:         deleteDatadogCanaryMonitor.App,
				picchu.LabelMonitorType: MonitorTypeCanary,
			},
		},
	}
	cli := fakeClient(sl)

	assert.NoError(deleteDatadogCanaryMonitor.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, sl)
}
