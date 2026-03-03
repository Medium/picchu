package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetScaleDefaults(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scale    ScaleInfo
		expected int32
	}{
		{
			name: "Default 0 with Min 16 uses Min",
			scale: ScaleInfo{
				Min:     int32Ptr(16),
				Default: 0,
				Max:     96,
			},
			expected: 16,
		},
		{
			name: "Default 0 with Min 1 uses Min",
			scale: ScaleInfo{
				Min:     int32Ptr(1),
				Default: 0,
				Max:     1,
			},
			expected: 1,
		},
		{
			name: "Default 0 with Min nil uses 1",
			scale: ScaleInfo{
				Min:     nil,
				Default: 0,
				Max:     0,
			},
			expected: 1,
		},
		{
			name: "Default 0 with Min 0 uses 1",
			scale: ScaleInfo{
				Min:     int32Ptr(0),
				Default: 0,
				Max:     0,
			},
			expected: 1,
		},
		{
			name: "Default already set is unchanged",
			scale: ScaleInfo{
				Min:     int32Ptr(16),
				Default: 48,
				Max:     96,
			},
			expected: 48,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			SetScaleDefaults(&tc.scale)
			assert.Equal(t, tc.expected, tc.scale.Default)
		})
	}
}

func int32Ptr(v int32) *int32 {
	return &v
}
