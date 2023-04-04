package plan

import (
	"context"
	_ "runtime"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"

	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteCanaryRules(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteCanaryRules := &DeleteCanaryRules{
		App:       "testapp",
		Namespace: "testnamespace",
		Tag:       "tag",
	}
	pr := &monitoring.PrometheusRule{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:      deleteCanaryRules.App,
				picchu.LabelTag:      deleteCanaryRules.Tag,
				picchu.LabelRuleType: RuleTypeCanary,
			},
		},
	}
	cli := fakeClient(pr)

	assert.NoError(deleteCanaryRules.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, pr)
}
