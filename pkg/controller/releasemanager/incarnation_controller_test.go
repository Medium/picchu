package releasemanager

import (
	T "testing"

	"go.medium.engineering/picchu/pkg/test"

	"github.com/stretchr/testify/assert"
)

func TestDivideReplicas(t *T.T) {
	log := test.MustNewLogger()
	out := IncarnationController{
		fleetSize: 4,
		log:       log,
	}
	// 5 are required, so each cluster should get 2 for a total of 8
	// since 1 would be 4 and 4 < 5
	assert.Equal(t, int32(2), out.divideReplicas(5, 100))
	assert.Equal(t, int32(1), out.divideReplicas(5, 80))
	assert.Equal(t, int32(2), out.divideReplicas(5, 81))
}
