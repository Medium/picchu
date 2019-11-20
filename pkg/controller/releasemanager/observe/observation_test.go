package observe

import (
	"testing"

	"go.medium.engineering/picchu/pkg/test"

	"github.com/stretchr/testify/assert"
)

func TestCombinedDeployedObservations(t *testing.T) {
	log := test.MustNewLogger()
	master := &Observation{}
	master = master.combine(&Observation{
		Clusters: []Cluster{{
			Name: "production-lime-a",
			Live: true,
		}},
		ReplicaSets: []replicaSet{{
			Desired: 2,
			Current: 1,
			Tag:     "tag-a",
			Cluster: Cluster{
				Name: "production-lime-a",
				Live: true,
			},
		}},
	})
	master = master.combine(&Observation{
		Clusters: []Cluster{{
			Name: "production-lime-b",
			Live: true,
		}},
		ReplicaSets: []replicaSet{{
			Desired: 3,
			Current: 2,
			Tag:     "tag-a",
			Cluster: Cluster{
				Name: "production-lime-b",
				Live: true,
			},
		}},
	})
	log.Info("Created OUT", "Observation", master)
	assert.NotNil(t, master.ForTag("tag-a"), "Should find info for tag")
	assert.Equal(t, 5, int(master.ForTag("tag-a").Stats.Desired.Sum), "Combined desired count should be accumlated")
	assert.Equal(t, 3, int(master.ForTag("tag-a").Stats.Current.Sum), "Combined current count should be accumlated")
	assert.True(t, master.ForTag("tag-a").Stats.Deployed, "Combined should be state deployed")
}

func TestCombinedNotDeployedObservations(t *testing.T) {
	log := test.MustNewLogger()
	master := &Observation{}
	master = master.combine(&Observation{
		Clusters: []Cluster{{
			Name: "production-lime-a",
			Live: true,
		}},
		ReplicaSets: []replicaSet{{
			Desired: 2,
			Current: 1,
			Tag:     "tag-a",
			Cluster: Cluster{
				Name: "production-lime-a",
				Live: true,
			},
		}},
	})
	master = master.combine(&Observation{
		Clusters: []Cluster{{
			Name: "production-lime-b",
			Live: true,
		}},
		ReplicaSets: []replicaSet{{
			Desired: 1,
			Current: 0,
			Tag:     "tag-a",
			Cluster: Cluster{
				Name: "production-lime-b",
				Live: true,
			},
		}},
	})
	log.Info("Created OUT", "Observation", master)
	assert.NotNil(t, master.ForTag("tag-a"), "Should find info for tag")
	assert.Equal(t, 3, int(master.ForTag("tag-a").Stats.Desired.Sum), "Combined desired count should be accumlated")
	assert.Equal(t, 1, int(master.ForTag("tag-a").Stats.Current.Sum), "Combined current count should be accumlated")
	assert.False(t, master.ForTag("tag-a").Stats.Deployed, "Combined should be state deployed")
}

func TestIntStat(t *testing.T) {
	stat := IntStat{}
	for _, n := range []int32{3, 5, 2, 4, 10, 5} {
		stat.Record(n)
	}
	assert.Equal(t, 6, int(stat.Count))
	assert.Equal(t, 29, int(stat.Sum))
	assert.Equal(t, 2, int(stat.Min))
	assert.Equal(t, 10, int(stat.Max))
}
