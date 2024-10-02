package plan_test

import (
	"context"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"

	"go.medium.engineering/picchu/mocks"
	planmocks "go.medium.engineering/picchu/plan/mocks"
	"go.medium.engineering/picchu/test"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	scalingFactor = "1.0"
	cluster       = &picchuv1alpha1.Cluster{
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactorString: &scalingFactor,
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
