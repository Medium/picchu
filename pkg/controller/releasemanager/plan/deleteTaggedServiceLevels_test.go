package plan

import (
	"context"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	_ "runtime"
	"testing"

	slo "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteTaggedServiceLevels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	deleteTaggedServiceLevels := &DeleteTaggedServiceLevels{
		App:       "testapp",
		Namespace: "testnamespace",
		Target:    "target",
		Tag:       "v1",
	}
	sl := &slo.ServiceLevel{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:    deleteTaggedServiceLevels.App,
				picchu.LabelTag:    deleteTaggedServiceLevels.Tag,
				picchu.LabelTarget: deleteTaggedServiceLevels.Target,
			},
		},
	}
	cli := fakeClient(sl)

	assert.NoError(deleteTaggedServiceLevels.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, sl)
}
