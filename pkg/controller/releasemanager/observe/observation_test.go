package observe

import (
	"testing"

	"go.medium.engineering/picchu/pkg/test"

	"github.com/stretchr/testify/assert"
)

func TestCombinedDeployedObservations(t *testing.T) {
	log := test.MustNewLogger()
	master := &Observation{}
	master = master.Combine(&Observation{
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
	master = master.Combine(&Observation{
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
	master = master.Combine(&Observation{
		ReplicaSets: []replicaSet{{
			Desired: 3,
			Current: 2,
			Tag:     "tag-a",
			Cluster: Cluster{
				Name: "production-moss-b",
				Live: false,
			},
		}},
	})
	log.Info("Created OUT", "Observation", master)
	info := master.InfoForTag("tag-a")
	assert.NotNil(t, info, "Should find info for tag")
	assert.Equal(t, 5, int(info.Live.Desired.Sum), "Combined desired count should be accumlated")
	assert.Equal(t, 3, int(info.Live.Current.Sum), "Combined current count should be accumlated")
	assert.Equal(t, 3, int(info.Standby.Desired.Sum), "Combined desired count should be accumlated")
	assert.Equal(t, 2, int(info.Standby.Current.Sum), "Combined current count should be accumlated")
	assert.Equal(t, 2, int(info.Live.Current.Count), "Correct number of records found")
	assert.Equal(t, 1, int(info.Standby.Current.Count), "Correct number of records found")
}

func TestCombinedNotDeployedObservations(t *testing.T) {
	log := test.MustNewLogger()
	master := &Observation{}
	master = master.Combine(&Observation{
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
	master = master.Combine(&Observation{
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
	master = master.Combine(&Observation{
		ReplicaSets: []replicaSet{{
			Desired: 1,
			Current: 0,
			Tag:     "tag-a",
			Cluster: Cluster{
				Name: "production-moss-b",
				Live: false,
			},
		}},
	})
	log.Info("Created OUT", "Observation", master)
	info := master.InfoForTag("tag-a")
	assert.NotNil(t, info, "Should find info for tag")
	assert.Equal(t, 3, int(info.Live.Desired.Sum), "Combined desired count should be accumlated")
	assert.Equal(t, 1, int(info.Live.Current.Sum), "Combined current count should be accumlated")
	assert.Equal(t, 1, int(info.Standby.Desired.Sum), "Combined desired count should be accumlated")
	assert.Equal(t, 0, int(info.Standby.Current.Sum), "Combined current count should be accumlated")
	assert.Equal(t, 2, int(info.Live.Current.Count), "Correct number of records found")
	assert.Equal(t, 1, int(info.Standby.Current.Count), "Correct number of records found")
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
