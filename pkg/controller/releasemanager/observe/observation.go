package observe

// DeploymentInfo is all release-relevent info about deployment status
type DeploymentInfo struct {
	Stats        DeploymentStats
	StandbyStats DeploymentStats
}

type DeploymentStats struct {
	Deployed bool
	Desired  IntStat
	Current  IntStat
}

type IntStat struct {
	Min   int32
	Max   int32
	Sum   int32
	Count int32
}

func (s *IntStat) Record(n int32) {
	if s.Count == 0 || s.Min > n {
		s.Min = n
	}
	if s.Max < n {
		s.Max = n
	}
	s.Sum += n
	s.Count += 1
}

type replicaSet struct {
	Desired int32
	Current int32
	Tag     string
	Cluster Cluster
}

// Observation encapsulates observed deployment state for an app target
type Observation struct {
	Clusters    []Cluster
	ReplicaSets []replicaSet
}

type Cluster struct {
	Name string
	Live bool
}

func (o *Observation) AddCluster(cluster Cluster) {
	for i := range o.Clusters {
		visit := o.Clusters[i]
		if visit.Name == cluster.Name && visit.Live == cluster.Live {
			return
		}
	}
	o.Clusters = append(o.Clusters, cluster)
}

func (o *Observation) LiveCount() int {
	cnt := 0
	for i := range o.Clusters {
		if o.Clusters[i].Live {
			cnt += 1
		}
	}
	return cnt
}

func (o *Observation) StandbyCount() int {
	cnt := 0
	for i := range o.Clusters {
		if !o.Clusters[i].Live {
			cnt += 1
		}
	}
	return cnt
}

func (o *Observation) combine(other *Observation) *Observation {
	observation := &Observation{
		ReplicaSets: append(o.ReplicaSets, other.ReplicaSets...),
	}
	for i := range o.Clusters {
		cluster := o.Clusters[i]
		observation.AddCluster(Cluster{Name: cluster.Name, Live: cluster.Live})
	}
	for i := range other.Clusters {
		cluster := other.Clusters[i]
		observation.AddCluster(Cluster{Name: cluster.Name, Live: cluster.Live})
	}
	return observation
}

// ForTag returns deployment info for a particular tag
func (o *Observation) ForTag(tag string) *DeploymentInfo {
	deployedClusters := map[string]bool{}
	deployedStandbyClusters := map[string]bool{}
	var di *DeploymentInfo

	for _, rs := range o.ReplicaSets {
		if rs.Tag != tag {
			continue
		}
		if di == nil {
			di = &DeploymentInfo{}
		}
		switch rs.Cluster.Live {
		case true:
			di.Stats.Desired.Record(rs.Desired)
			di.Stats.Current.Record(rs.Current)
			if rs.Current > 0 {
				deployedClusters[rs.Cluster.Name] = true
			}
		default:
			di.StandbyStats.Desired.Record(rs.Desired)
			di.StandbyStats.Current.Record(rs.Current)
			if rs.Current > 0 {
				deployedStandbyClusters[rs.Cluster.Name] = true
			}
		}
	}

	if di != nil && len(deployedClusters) == o.LiveCount() {
		di.Stats.Deployed = true
	}
	if di != nil && len(deployedStandbyClusters) == o.StandbyCount() {
		di.StandbyStats.Deployed = true
	}
	return di
}
