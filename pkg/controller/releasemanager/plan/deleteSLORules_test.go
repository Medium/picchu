package plan

import (
	"context"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	_ "runtime"
	"testing"

	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteSLORules(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteSLORules := &DeleteSLORules{
		App:       "testapp",
		Namespace: "testnamespace",
	}
	pr := &monitoring.PrometheusRule{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:      deleteSLORules.App,
				picchu.LabelRuleType: RuleTypeSLO,
			},
		},
	}
	cli := fakeClient(pr)

	assert.NoError(deleteSLORules.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, pr)
}
