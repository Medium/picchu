package releasemanager

import (
	T "testing"

	"go.medium.engineering/picchu/pkg/test"

	"github.com/stretchr/testify/assert"
)

func TestDivideReplicas(t *T.T) {
	log := test.MustNewLogger()
	out := IncarnationController{
		clusterInfo: ClusterInfoList{{
			Name:          "cluster-0",
			Live:          true,
			ScalingFactor: 1.0,
		}, {
			Name:          "cluster-1",
			Live:          true,
			ScalingFactor: 1.0,
		}, {
			Name:          "cluster-2",
			Live:          true,
			ScalingFactor: 1.0,
		}, {
			Name:          "cluster-3",
			Live:          true,
			ScalingFactor: 1.0,
		}},
		log: log,
	}
	// 5 are required, so each cluster should get 2 for a total of 8
	// since 1 would be 4 and 4 < 5
	assert.Equal(t, int32(2), out.divideReplicas(5, 100))
	assert.Equal(t, int32(1), out.divideReplicas(5, 80))
	assert.Equal(t, int32(2), out.divideReplicas(5, 81))
	assert.Equal(t, int32(8), out.expectedTotalReplicas(5, 81))
	assert.Equal(t, int32(8), out.expectedTotalReplicas(5, 81))
	assert.Equal(t, int32(8), out.expectedTotalReplicas(5, 81))
}

func TestDivideReplicasWithScaling(t *T.T) {
	log := test.MustNewLogger()
	out := IncarnationController{
		clusterInfo: ClusterInfoList{{
			Name:          "cluster-0",
			Live:          true,
			ScalingFactor: 2.0,
		}, {
			Name:          "cluster-1",
			Live:          true,
			ScalingFactor: 2.0,
		}, {
			Name:          "cluster-2",
			Live:          true,
			ScalingFactor: 3.0,
		}, {
			Name:          "cluster-3",
			Live:          true,
			ScalingFactor: 1.0,
		}},
		log: log,
	}
	// 5 are required, so each cluster should get 2 for a total of 8
	// since 1 would be 4 and 4 < 5
	assert.Equal(t, int32(2), out.divideReplicas(5, 100))
	assert.Equal(t, int32(1), out.divideReplicas(5, 80))
	assert.Equal(t, int32(2), out.divideReplicas(5, 81))
	assert.Equal(t, int32(16), out.expectedTotalReplicas(5, 81))
	assert.Equal(t, int32(16), out.expectedTotalReplicas(5, 81))
	assert.Equal(t, int32(16), out.expectedTotalReplicas(5, 81))
}
