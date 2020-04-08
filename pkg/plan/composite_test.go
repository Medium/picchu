package plan_test

import (
	"context"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	"testing"

	"go.medium.engineering/picchu/pkg/mocks"
	planmocks "go.medium.engineering/picchu/pkg/plan/mocks"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	scalingFactor = 1.0
	cluster       = &picchuv1alpha1.Cluster{
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &scalingFactor,
		},
	}
)

func TestCompositePlanFlat(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	p1 := planmocks.NewMockPlan(ctrl)
	p2 := planmocks.NewMockPlan(ctrl)
	cli := mocks.NewMockClient(ctrl)

	ctx := context.TODO()

	p1.
		EXPECT().
		Apply(ctx, cli, cluster, gomock.Any()).
		Return(nil).
		Times(1)

	p2.
		EXPECT().
		Apply(ctx, cli, cluster, gomock.Any()).
		Return(nil).
		Times(1)

	composite := plan.All(p1, p2)
	assert.NoError(t, composite.Apply(ctx, cli, cluster, log), "no error")
}

func TestCompositePlanLevels(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	p1 := planmocks.NewMockPlan(ctrl)
	p2 := planmocks.NewMockPlan(ctrl)
	p3 := planmocks.NewMockPlan(ctrl)
	cli := mocks.NewMockClient(ctrl)

	ctx := context.TODO()

	p1.
		EXPECT().
		Apply(ctx, cli, cluster, gomock.Any()).
		Return(nil).
		Times(1)
	p2.
		EXPECT().
		Apply(ctx, cli, cluster, gomock.Any()).
		Return(nil).
		Times(1)
	p3.
		EXPECT().
		Apply(ctx, cli, cluster, gomock.Any()).
		Return(nil).
		Times(1)

	composite := plan.All(p1, p2)
	composite = plan.All(composite, p3)
	assert.NoError(t, composite.Apply(ctx, cli, cluster, log), "no error")
}
