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

func TestDeleteDatadogSLOs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteDatadogSLOs := &DeleteDatadogSLOs{
		App:       "echo",
		Target:    "prod",
		Namespace: "datadog",
		Tag:       "main-123",
	}
	sl := &ddog.DatadogSLO{
		ObjectMeta: meta.ObjectMeta{
			Name:      "slo1",
			Namespace: "datadog",
			Labels: map[string]string{
				picchu.LabelApp:    deleteDatadogSLOs.App,
				picchu.LabelTag:    deleteDatadogSLOs.Tag,
				picchu.LabelTarget: deleteDatadogSLOs.Target,
			},
		},
	}
	cli := fakeClient(sl)

	assert.NoError(deleteDatadogSLOs.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, sl)
}
