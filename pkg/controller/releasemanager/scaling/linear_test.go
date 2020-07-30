package scaling

import (
	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/test"
	"k8s.io/utils/pointer"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling/mocks"
)

func TestLinearScaling(t *testing.T) {
	log := test.MustNewLogger()

	for _, test := range []struct {
		Name             string
		CanRamp          bool
		CurrentPercent   uint32
		PeakPercent      uint32
		Increment        uint32
		Max              uint32
		Delay            int64
		LastUpdated      time.Time
		RemainingPercent uint32
		Expected         uint32
	}{
		{
			Name:             "IncFromZero",
			CanRamp:          true,
			CurrentPercent:   0,
			PeakPercent:      0,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         5,
		},
		{
			Name:             "DontIncAboveMax100",
			CanRamp:          true,
			CurrentPercent:   100,
			PeakPercent:      100,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         100,
		},
		{
			Name:             "DontIncAboveRemaining",
			CanRamp:          true,
			CurrentPercent:   50,
			PeakPercent:      100,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 50,
			Expected:         50,
		},
		{
			Name:             "DontIncAboveMax50",
			CanRamp:          true,
			CurrentPercent:   50,
			PeakPercent:      100,
			Increment:        5,
			Max:              50,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         50,
		},
		{
			Name:             "IncFrom50",
			CanRamp:          true,
			CurrentPercent:   50,
			PeakPercent:      50,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         55,
		},
		{
			Name:             "DontIncIfUnreconciled",
			CanRamp:          false,
			CurrentPercent:   50,
			PeakPercent:      50,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         50,
		},
		{
			Name:             "DontIncToPeak",
			CanRamp:          true,
			CurrentPercent:   0,
			PeakPercent:      99,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 100,
			Expected:         5,
		},
		{
			Name:             "DontIncToMaxRemaining",
			CanRamp:          true,
			CurrentPercent:   0,
			PeakPercent:      99,
			Increment:        5,
			Max:              100,
			Delay:            5,
			LastUpdated:      time.Time{},
			RemainingPercent: 80,
			Expected:         5,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert := testify.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockScalableTarget(ctrl)
			releaseInfo := picchu.ReleaseInfo{
				Eligible:         true,
				Max:              test.Max,
				ScalingStrategy:  picchu.ScalingStrategyLinear,
				GeometricScaling: picchu.GeometricScaling{},
				LinearScaling:    picchu.LinearScaling{},
				Rate: picchu.RateInfo{
					Increment:    test.Increment,
					DelaySeconds: pointer.Int64Ptr(test.Delay),
				},
				Schedule: "always",
				TTL:      86400,
			}

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

			assert.Equal(test.Expected, LinearScale(m, test.RemainingPercent, time.Now(), log))
		})
	}
}
