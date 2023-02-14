package plan

import (
	"context"
	_ "runtime"
	"testing"

	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	ktest "go.medium.engineering/kubernetes/pkg/test"

	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteServiceMonitors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteServiceMonitors := &DeleteServiceMonitors{
		App:       "testapp",
		Namespace: "testnamespace",
	}
	sl := &monitoring.ServiceMonitor{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp: deleteServiceMonitors.App,
			},
		},
	}
	cli := fakeClient(sl)

	assert.NoError(deleteServiceMonitors.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, sl)
}
