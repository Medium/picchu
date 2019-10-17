package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDeleteApp(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteapp := &DeleteApp{
		Namespace: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(nil).
		Times(1)

	assert.NoError(t, deleteapp.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}

func TestDeleteAlreadyDeletedApp(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteapp := &DeleteApp{
		Namespace: "testnamespace",
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("", "testnamespace"), mocks.Kind("Namespace"))).
		Return(common.NotFoundError).
		Times(1)

	assert.NoError(t, deleteapp.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
