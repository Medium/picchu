package plan

import (
	"context"
	_ "runtime"
	"testing"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/mocks"
	"go.medium.engineering/picchu/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTaggedDeleteServiceLevels(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteTaggedServiceLevels := &DeleteTaggedServiceLevels{
		App:       "testapp",
		Namespace: "testnamespace",
		Target:    "target",
		Tag:       "v1",
	}
	ctx := context.TODO()

	opts := &client.ListOptions{
		Namespace: deleteTaggedServiceLevels.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:    deleteTaggedServiceLevels.App,
			picchuv1alpha1.LabelTag:    deleteTaggedServiceLevels.Tag,
			picchuv1alpha1.LabelTarget: deleteTaggedServiceLevels.Target,
		}),
	}

	sl := []slov1alpha1.ServiceLevel{
		slov1alpha1.ServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "testnamespace",
			},
		},
	}

	m.
		EXPECT().
		List(ctx, mocks.InjectServiceLevels(sl), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("testnamespace", "test"), mocks.Kind("ServiceLevel"))).
		Return(nil).
		Times(1)

	assert.NoError(t, deleteTaggedServiceLevels.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
