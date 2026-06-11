package plan

import (
	"context"
	_ "runtime"
	"testing"

	ktest "go.medium.engineering/kubernetes/pkg/test"

	slo "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestDeleteServiceLevels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteServiceLevels := &DeleteServiceLevels{
		App:       "testapp",
		Namespace: "testnamespace",
		Target:    "target",
	}
	shared := &slo.PrometheusServiceLevel{
		ObjectMeta: meta.ObjectMeta{
			Name:      "testapp-target-servicelevels",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:    deleteServiceLevels.App,
				picchu.LabelTarget: deleteServiceLevels.Target,
			},
		},
	}
	// Legacy per-tag PSLs share the {app, target} labels and are swept too.
	legacy := &slo.PrometheusServiceLevel{
		ObjectMeta: meta.ObjectMeta{
			Name:      "testapp-target-v123-servicelevels",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:    deleteServiceLevels.App,
				picchu.LabelTarget: deleteServiceLevels.Target,
				picchu.LabelTag:    "v123",
			},
		},
	}
	otherApp := &slo.PrometheusServiceLevel{
		ObjectMeta: meta.ObjectMeta{
			Name:      "otherapp-target-servicelevels",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:    "otherapp",
				picchu.LabelTarget: deleteServiceLevels.Target,
			},
		},
	}
	cli := fakeClient(shared, legacy, otherApp)

	assert.NoError(deleteServiceLevels.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, shared)
	ktest.AssertNotFound(ctx, t, cli, legacy)
	assert.NoError(cli.Get(ctx, types.NamespacedName{Name: otherApp.Name, Namespace: otherApp.Namespace}, &slo.PrometheusServiceLevel{}), "Other app's PSL should survive.")
}
