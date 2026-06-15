package controllers

import (
	"context"
	tt "testing"

	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	rmplan "go.medium.engineering/picchu/controllers/plan"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/mocks"
	"go.medium.engineering/picchu/plan"
	"go.medium.engineering/picchu/test"
	"go.uber.org/mock/gomock"
)

// recordingApplier captures the plans applied by ResourceSyncer so tests can
// assert on the sync-vs-delete lifecycle decision without a cluster.
type recordingApplier struct {
	plans []plan.Plan
}

func (a *recordingApplier) Apply(_ context.Context, p plan.Plan) error {
	a.plans = append(a.plans, p)
	return nil
}

func createTestIncarnationWithSLOs(tag string, currentState State, currentPercent int, slos []*picchuv1alpha1.SlothServiceLevelObjective) *Incarnation {
	i := createTestIncarnation(tag, currentState, currentPercent)
	i.revision.Spec.Targets[0].SlothServiceLevelObjectives = slos
	i.revision.Spec.Targets[0].ServiceLevelObjectiveLabels = picchuv1alpha1.ServiceLevelObjectiveLabels{
		ServiceLevelLabels: map[string]string{"severity": "test"},
	}
	return i
}

func TestPrepareServiceLevelObjectives(t *tt.T) {
	enabledSLO := &picchuv1alpha1.SlothServiceLevelObjective{Enabled: true, Name: "availability"}
	disabledSLO := &picchuv1alpha1.SlothServiceLevelObjective{Enabled: false, Name: "latency"}

	mixed := createTestIncarnationWithSLOs("mixed", releasing, 100,
		[]*picchuv1alpha1.SlothServiceLevelObjective{enabledSLO, disabledSLO})
	allDisabled := createTestIncarnationWithSLOs("disabled", releasing, 100,
		[]*picchuv1alpha1.SlothServiceLevelObjective{disabledSLO})

	testcases := []struct {
		name         string
		deployed     []*Incarnation
		releasable   []*Incarnation
		expectedSLOs []string
	}{
		{
			name:         "filters Enabled=false at the source",
			deployed:     []*Incarnation{mixed},
			releasable:   []*Incarnation{mixed},
			expectedSLOs: []string{"availability"},
		},
		{
			name:         "all SLOs disabled yields empty (drives the delete branch)",
			deployed:     []*Incarnation{allDisabled},
			releasable:   []*Incarnation{allDisabled},
			expectedSLOs: nil,
		},
		{
			name:         "no deployed incarnations yields empty",
			deployed:     nil,
			releasable:   nil,
			expectedSLOs: nil,
		},
		{
			name:         "deployed but nothing releasable yields empty",
			deployed:     []*Incarnation{mixed},
			releasable:   nil,
			expectedSLOs: nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *tt.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := NewMockIncarnations(ctrl)
			m.EXPECT().deployed().Return(tc.deployed).AnyTimes()
			m.EXPECT().releasable().Return(tc.releasable).AnyTimes()

			r := &ResourceSyncer{incarnations: m, log: test.MustNewLogger()}
			slos, _ := r.prepareServiceLevelObjectives()

			var names []string
			for _, slo := range slos {
				names = append(names, slo.Name)
			}
			assert.Equal(t, tc.expectedSLOs, names)
		})
	}
}

func TestSyncServiceLevelsLifecycle(t *tt.T) {
	enabledSLO := &picchuv1alpha1.SlothServiceLevelObjective{Enabled: true, Name: "availability"}
	disabledSLO := &picchuv1alpha1.SlothServiceLevelObjective{Enabled: false, Name: "latency"}

	testcases := []struct {
		name      string
		namespace string
		slos      []*picchuv1alpha1.SlothServiceLevelObjective
		// expected plan types in apply order
		expected []plan.Plan
	}{
		{
			name:      "enabled SLOs present syncs the shared PSL",
			namespace: "service-levels",
			slos:      []*picchuv1alpha1.SlothServiceLevelObjective{enabledSLO, disabledSLO},
			expected:  []plan.Plan{&rmplan.EnsureNamespace{}, &rmplan.SyncServiceLevels{}},
		},
		{
			name:      "no enabled SLOs deletes the shared PSL",
			namespace: "service-levels",
			slos:      []*picchuv1alpha1.SlothServiceLevelObjective{disabledSLO},
			expected:  []plan.Plan{&rmplan.DeleteServiceLevels{}},
		},
		{
			name:      "no service-levels namespace configured is a no-op",
			namespace: "",
			slos:      []*picchuv1alpha1.SlothServiceLevelObjective{enabledSLO},
			expected:  nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *tt.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			inc := createTestIncarnationWithSLOs("inc", releasing, 100, tc.slos)
			m := NewMockIncarnations(ctrl)
			m.EXPECT().deployed().Return([]*Incarnation{inc}).AnyTimes()
			m.EXPECT().releasable().Return([]*Incarnation{inc}).AnyTimes()

			applier := &recordingApplier{}
			r := &ResourceSyncer{
				incarnations: m,
				planApplier:  applier,
				log:          test.MustNewLogger(),
				picchuConfig: utils.Config{ServiceLevelsNamespace: tc.namespace},
				instance: &picchuv1alpha1.ReleaseManager{
					Spec: picchuv1alpha1.ReleaseManagerSpec{App: "testapp", Target: "production"},
				},
			}

			assert.NoError(t, r.syncServiceLevels(context.Background()))

			assert.Len(t, applier.plans, len(tc.expected))
			for i := range tc.expected {
				assert.IsType(t, tc.expected[i], applier.plans[i])
			}

			for _, p := range applier.plans {
				switch typed := p.(type) {
				case *rmplan.SyncServiceLevels:
					assert.Equal(t, "testapp", typed.App)
					assert.Equal(t, "production", typed.Target)
					assert.Equal(t, tc.namespace, typed.Namespace)
					// Only enabled SLOs reach the plan.
					for _, slo := range typed.ServiceLevelObjectives {
						assert.True(t, slo.Enabled)
					}
				case *rmplan.DeleteServiceLevels:
					assert.Equal(t, "testapp", typed.App)
					assert.Equal(t, "production", typed.Target)
					assert.Equal(t, tc.namespace, typed.Namespace)
				}
			}
		})
	}
}

// The shared PSL has no OwnerReference (it lives in the cross-app
// service-levels namespace), so the final reconcile MUST delete it before the
// ReleaseManager deletes itself — afterwards no controller will ever look at
// this app/target again. Regression test for the orphan path.
func TestSyncDeletesSharedServiceLevelsOnTeardown(t *tt.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	m := NewMockIncarnations(ctrl)
	m.EXPECT().revisioned().Return([]*Incarnation{}).Times(1)
	m.EXPECT().deployed().Return(nil).AnyTimes()

	instance := &picchuv1alpha1.ReleaseManager{
		Spec: picchuv1alpha1.ReleaseManagerSpec{App: "testapp", Target: "production"},
	}

	cli := mocks.NewMockClient(ctrl)
	deleted := false
	cli.
		EXPECT().
		Delete(ctx, instance).
		DoAndReturn(func(_ context.Context, _ interface{}, _ ...interface{}) error {
			deleted = true
			return nil
		}).
		Times(1)

	applier := &recordingApplier{}
	r := &ResourceSyncer{
		deliveryClient: cli,
		planApplier:    applier,
		incarnations:   m,
		instance:       instance,
		log:            test.MustNewLogger(),
		picchuConfig:   utils.Config{ServiceLevelsNamespace: "service-levels"},
	}

	_, err := r.sync(ctx)
	assert.NoError(t, err)
	assert.True(t, deleted, "ReleaseManager should self-delete on teardown")

	// The shared PSL delete must have been applied before the RM disappears.
	if assert.Len(t, applier.plans, 1) {
		del, ok := applier.plans[0].(*rmplan.DeleteServiceLevels)
		if assert.True(t, ok, "expected DeleteServiceLevels, got %T", applier.plans[0]) {
			assert.Equal(t, "testapp", del.App)
			assert.Equal(t, "production", del.Target)
			assert.Equal(t, "service-levels", del.Namespace)
		}
	}
}
