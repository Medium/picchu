package plan

import (
	"context"
	_ "runtime"
	"testing"

	ktest "go.medium.engineering/kubernetes/pkg/test"

	slo "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	slov1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	sl := &slo.ServiceLevel{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:    deleteServiceLevels.App,
				picchu.LabelTarget: deleteServiceLevels.Target,
			},
		},
	}
	cli := fakeClient(sl)

	assert.NoError(deleteServiceLevels.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, sl)
}

func TestDeleteServiceLevelsSloth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteServiceLevels := &DeleteServiceLevels{
		App:       "testapp",
		Namespace: "testnamespace",
		Target:    "target",
	}
	sl := &slov1.PrometheusServiceLevel{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:    deleteServiceLevels.App,
				picchu.LabelTarget: deleteServiceLevels.Target,
			},
		},
	}
	cli := fakeClient(sl)

	assert.NoError(deleteServiceLevels.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, sl)
}
