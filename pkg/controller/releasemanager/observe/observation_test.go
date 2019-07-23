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
		Clusters: []string{"production-lime-a"},
		ReplicaSets: []replicaSet{{
			Desired: 2,
			Current: 1,
			Tag:     "tag-a",
			Cluster: "production-lime-a",
		}},
	})
	master = master.combine(&Observation{
		Clusters: []string{"production-lime-b"},
		ReplicaSets: []replicaSet{{
			Desired: 3,
			Current: 2,
			Tag:     "tag-a",
			Cluster: "production-lime-b",
		}},
	})
	log.Info("Created OUT", "Observation", master)
	assert.NotNil(t, master.ForTag("tag-a"), "Should find info for tag")
	assert.Equal(t, 5, int(master.ForTag("tag-a").Desired), "Combined desired count should be accumlated")
	assert.Equal(t, 3, int(master.ForTag("tag-a").Current), "Combined current count should be accumlated")
	assert.True(t, master.ForTag("tag-a").Deployed, "Combined should be state deployed")
}

func TestCombinedNotDeployedObservations(t *testing.T) {
	log := test.MustNewLogger()
	master := &Observation{}
	master = master.combine(&Observation{
		Clusters: []string{"production-lime-a"},
		ReplicaSets: []replicaSet{{
			Desired: 2,
			Current: 1,
			Tag:     "tag-a",
			Cluster: "production-lime-a",
		}},
	})
	master = master.combine(&Observation{
		Clusters: []string{"production-lime-b"},
		ReplicaSets: []replicaSet{{
			Desired: 1,
			Current: 0,
			Tag:     "tag-a",
			Cluster: "production-lime-b",
		}},
	})
	log.Info("Created OUT", "Observation", master)
	assert.NotNil(t, master.ForTag("tag-a"), "Should find info for tag")
	assert.Equal(t, 3, int(master.ForTag("tag-a").Desired), "Combined desired count should be accumlated")
	assert.Equal(t, 1, int(master.ForTag("tag-a").Current), "Combined current count should be accumlated")
	assert.False(t, master.ForTag("tag-a").Deployed, "Combined should be state deployed")
}
