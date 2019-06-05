package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/controller/releasemanager/mocks"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDeleteApp(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &DeleteApp{
		Namespace: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, log), "Shouldn't return error.")
}

func TestDeleteAlreadyDeletedApp(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &DeleteApp{
		Namespace: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(notFoundError).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, log), "Shouldn't return error.")
}
