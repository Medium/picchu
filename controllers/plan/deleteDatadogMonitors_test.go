package plan

import (
	"context"
	_ "runtime"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteDatadogMonitors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteDatadogMonitors := &DeleteDatadogMonitors{
		App:       "testapp",
		Namespace: "testnamespace",
	}
	pr := &ddog.DatadogMonitor{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:                 deleteDatadogMonitors.App,
				picchuv1alpha1.LabelMonitorType: MonitorTypeSLO,
			},
		},
	}
	cli := fakeClient(pr)

	assert.NoError(deleteDatadogMonitors.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, pr)
}
