package plan

import (
	"context"
	"go.medium.engineering/picchu/pkg/plan"
	"testing"

	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreatesNamespace(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	en := &EnsureNamespace{
		Name: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, gomock.Any(), mocks.Kind("Namespace")).
		Return(common.NotFoundError).
		Times(1)
	m.
		EXPECT().
		Create(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(nil).
		Times(1)

	assert.NoError(t, en.Apply(ctx, m, plan.Options{ScalingFactor: 1.0}, log), "Shouldn't return error.")
}

func TestIgnoreNamespace(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	en := &EnsureNamespace{
		Name: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, gomock.Any(), mocks.Kind("Namespace")).
		Return(nil).
		Times(1)

	assert.NoError(t, en.Apply(ctx, m, plan.Options{ScalingFactor: 1.0}, log), "Shouldn't return error.")
}

func TestUpdatesNamespace(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	en := &EnsureNamespace{
		Name: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, gomock.Any(), mocks.UpdateNamespaceLabels(map[string]string{})).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Update(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(nil).
		Times(1)

	assert.NoError(t, en.Apply(ctx, m, plan.Options{ScalingFactor: 1.0}, log), "Shouldn't return error.")
}
