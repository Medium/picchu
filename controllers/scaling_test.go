package controllers

import (
	"context"
	"math"
	stdtesting "testing"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	"go.medium.engineering/picchu/test"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeController struct {
	rm  *picchuv1alpha1.ReleaseManager
	log logr.Logger
}

func (f *fakeController) expectedTotalReplicas(count int32, percent int32) int32 {
	capped := percent
	if capped > 100 {
		capped = 100
	}
	factor := float64(capped) / 100.0
	return int32(math.Ceil(factor * float64(count)))
}

func (f *fakeController) applyPlan(ctx context.Context, name string, p plan.Plan) error {
	return nil
}

func (f *fakeController) divideReplicas(count int32, percent int32) int32 {
	return count
}

func (f *fakeController) getReleaseManager() *picchuv1alpha1.ReleaseManager {
	return f.rm
}

func (f *fakeController) getLog() logr.Logger {
	return f.log
}

func (f *fakeController) getConfigMaps(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	return nil, nil
}

func (f *fakeController) getSecrets(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	return nil, nil
}

func (f *fakeController) getExternalSecrets(ctx context.Context, opts *client.ListOptions) ([]es.ExternalSecret, error) {
	return nil, nil
}

func (f *fakeController) liveCount() int {
	return 1
}

func (f *fakeController) getTotalCurrentCapacity(excludeTag string) int32 {
	return 0
}

func TestCanRampTo_UsesPeakWhenUnscaledRevision(t *stdtesting.T) {
	log := test.MustNewLogger()
	min := int32(1)

	rm := &picchuv1alpha1.ReleaseManager{
		Spec: picchuv1alpha1.ReleaseManagerSpec{
			App:    "app",
			Target: "prod",
		},
		Status: picchuv1alpha1.ReleaseManagerStatus{
			Revisions: []picchuv1alpha1.ReleaseManagerRevisionStatus{
				{
					Tag:            "old",
					CurrentPercent: 50,
					PeakPercent:    100,
					Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{
						Current: 3,
						Peak:    10,
					},
				},
			},
		},
	}

	controller := &fakeController{rm: rm, log: log}
	revision := &picchuv1alpha1.Revision{
		Spec: picchuv1alpha1.RevisionSpec{
			App: picchuv1alpha1.RevisionApp{Name: "app"},
			Targets: []picchuv1alpha1.RevisionTarget{
				{
					Name: "prod",
					Scale: picchuv1alpha1.ScaleInfo{
						Min: &min,
					},
				},
			},
		},
	}

	inc := Incarnation{
		controller: controller,
		tag:        "new",
		revision:   revision,
		log:        log,
		status: &picchuv1alpha1.ReleaseManagerRevisionStatus{
			Scale: picchuv1alpha1.ReleaseManagerRevisionScaleStatus{
				Current: 5,
			},
		},
	}

	sta := ScalableTargetAdapter{inc}
	if sta.CanRampTo(60) {
		t.Fatalf("expected CanRampTo to be false when peak capacity is required")
	}
}
