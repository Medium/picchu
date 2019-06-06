package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/controller/releasemanager/mocks"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreatesNamespace(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &EnsureNamespace{
		Name: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, gomock.Any(), mocks.Kind("Namespace")).
		Return(notFoundError).
		Times(1)
	m.
		EXPECT().
		Create(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, log), "Shouldn't return error.")
}

func TestIngoreNamespace(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &EnsureNamespace{
		Name: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, gomock.Any(), mocks.Kind("Namespace")).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, log), "Shouldn't return error.")
}

func TestUpdatesNamespace(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &EnsureNamespace{
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

	assert.NoError(t, plan.Apply(ctx, m, log), "Shouldn't return error.")
}
