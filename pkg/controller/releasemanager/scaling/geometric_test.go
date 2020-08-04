package scaling

import (
	"github.com/golang/mock/gomock"
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling/mocks"
	"go.medium.engineering/picchu/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestGeometricScaling(t *testing.T) {
	log := test.MustNewLogger()

	for _, test := range []struct {
		Name             string
		CanRamp          bool
		CurrentPercent   uint32
		PeakPercent      uint32
		Factor           uint32
		Max              uint32
		Start            uint32
		Delay            time.Duration
		LastUpdated      time.Time
		RemainingPercent uint32
		Expected         uint32
	}{
		{
			Name:             "StartAt5",
			CanRamp:          true,
			CurrentPercent:   0,
			PeakPercent:      0,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         5,
		},
		{
			Name:             "DontIncAboveMax",
			CanRamp:          true,
			CurrentPercent:   100,
			PeakPercent:      100,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         100,
		},
		{
			Name:             "DontIncAboveRemaining",
			CanRamp:          true,
			CurrentPercent:   50,
			PeakPercent:      100,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 50,
			Expected:         50,
		},
		{
			Name:             "DontIncAboveMax50",
			CanRamp:          true,
			CurrentPercent:   50,
			PeakPercent:      100,
			Factor:           2,
			Max:              50,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         50,
		},
		{
			Name:             "IncMyFactor25",
			CanRamp:          true,
			CurrentPercent:   25,
			PeakPercent:      25,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         50,
		},
		{
			Name:             "IncMyFactor50",
			CanRamp:          true,
			CurrentPercent:   50,
			PeakPercent:      50,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         100,
		},
		{
			Name:             "DontIncIfUnreconciled",
			CanRamp:          false,
			CurrentPercent:   50,
			PeakPercent:      50,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         50,
		},
		{
			Name:             "DontSkipToPeak",
			CanRamp:          true,
			CurrentPercent:   0,
			PeakPercent:      99,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         5,
		},
		{
			Name:             "DontSkipToMaxRemaining",
			CanRamp:          true,
			CurrentPercent:   0,
			PeakPercent:      99,
			Factor:           2,
			Max:              100,
			Start:            5,
			Delay:            time.Duration(5) * time.Second,
			LastUpdated:      time.Time{},
			RemainingPercent: 80,
			Expected:         5,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			assert := testify.New(t)
			releaseInfo := picchu.ReleaseInfo{
				Eligible:        true,
				Max:             test.Max,
				ScalingStrategy: picchu.ScalingStrategyGeometric,
				GeometricScaling: picchu.GeometricScaling{
					Start:  test.Start,
					Factor: test.Factor,
					Delay:  &metav1.Duration{Duration: test.Delay},
				},
				LinearScaling: picchu.LinearScaling{},
				Schedule:      "always",
				TTL:           86400,
			}
			m := mocks.NewMockScalableTarget(ctrl)
			m.
				EXPECT().
				CurrentPercent().
				Return(test.CurrentPercent).
				AnyTimes()
			m.
				EXPECT().
				CanRampTo(gomock.Any()).
				Return(test.CanRamp).
				AnyTimes()
			m.
				EXPECT().
				PeakPercent().
				Return(test.PeakPercent).
				AnyTimes()
			m.
				EXPECT().
				ReleaseInfo().
				Return(releaseInfo).
				AnyTimes()
			m.
				EXPECT().
				LastUpdated().
				Return(test.LastUpdated).
				AnyTimes()

			assert.Equal(test.Expected, GeometricScale(m, test.RemainingPercent, time.Now(), log))
		})
	}
}
