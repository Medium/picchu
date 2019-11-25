package observe

// IntStat records int32 stats
type IntStat struct {
	Min   int32
	Max   int32
	Sum   int32
	Count int32
}

// Record updates IntStat with new information
func (s *IntStat) Record(n int32) {
	if s.Count == 0 || s.Min > n {
		s.Min = n
	}
	if s.Max < n {
		s.Max = n
	}
	s.Sum += n
	s.Count++
}

// DeploymentInfo is release-relevent info about deployment status
type DeploymentInfo struct {
	Desired IntStat
	Current IntStat
}

// DeploymentInfoSet returns deployment info for Live and Standby clusters
type DeploymentInfoSet struct {
	Live    *DeploymentInfo
	Standby *DeploymentInfo
}

// NewDeploymentInfoSet returns a new DeploymentInfoSet
func NewDeploymentInfoSet() *DeploymentInfoSet {
	return &DeploymentInfoSet{
		Live:    &DeploymentInfo{},
		Standby: &DeploymentInfo{},
	}
}

// GetInfo returns Info based on liveness flag
func (dis *DeploymentInfoSet) GetInfo(liveness bool) *DeploymentInfo {
	if liveness {
		return dis.Live
	}
	return dis.Standby
}

// Cluster represents a target cluster
type Cluster struct {
	Name string
	Live bool
}

// replicaSet represents a replicaset in a target cluster namespace
type replicaSet struct {
	Desired int32
	Current int32
	Tag     string
	Cluster Cluster
}

// Observation encapsulates observed deployment state for an app target
type Observation struct {
	ReplicaSets []replicaSet
}

// Count returns the number of unique live or standby clusters included in an Observation
func (o *Observation) Count(liveness bool) int {
	nameSet := map[string]bool{}

	for i := range o.ReplicaSets {
		cluster := o.ReplicaSets[i].Cluster
		if cluster.Live == liveness {
			nameSet[cluster.Name] = true
		}
	}
	return len(nameSet)
}

// Combine merges two Observations into a new Observation
func (o *Observation) Combine(other *Observation) *Observation {
	return &Observation{
		ReplicaSets: append(o.ReplicaSets, other.ReplicaSets...),
	}
}

// InfoForTag returns a deployment info set for a particular tag
func (o *Observation) InfoForTag(tag string) *DeploymentInfoSet {
	var dis *DeploymentInfoSet

	for _, rs := range o.ReplicaSets {
		if rs.Tag != tag {
			continue
		}
		if dis == nil {
			dis = NewDeploymentInfoSet()
		}
		di := dis.GetInfo(rs.Cluster.Live)
		di.Desired.Record(rs.Desired)
		di.Current.Record(rs.Current)
	}
	return dis
}
