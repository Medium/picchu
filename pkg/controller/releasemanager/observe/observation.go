package observe

// DeploymentInfo is all release-relevent info about deployment status
type DeploymentInfo struct {
	Deployed bool
	Desired  int32
	Current  int32
}

type replicaSet struct {
	Desired int32
	Current int32
	Tag     string
	Cluster string
}

// Observation encapsulates observed deployment state for an app target
type Observation struct {
	Clusters    []string
	ReplicaSets []replicaSet
}

func (o *Observation) combine(other *Observation) *Observation {
	clusterSet := map[string]bool{}
	for _, cluster := range o.Clusters {
		clusterSet[cluster] = true
	}
	for _, cluster := range other.Clusters {
		clusterSet[cluster] = true
	}
	clusters := []string{}
	for k, _ := range clusterSet {
		clusters = append(clusters, k)
	}
	return &Observation{
		Clusters:    clusters,
		ReplicaSets: append(o.ReplicaSets, other.ReplicaSets...),
	}
}

// ForTag returns deployment info for a particular tag
func (o *Observation) ForTag(tag string) *DeploymentInfo {
	deployedClusters := map[string]bool{}
	var di *DeploymentInfo

	for _, rs := range o.ReplicaSets {
		if rs.Tag != tag {
			continue
		}
		if di == nil {
			di = &DeploymentInfo{}
		}
		if rs.Current > 0 {
			deployedClusters[rs.Cluster] = true
		}
		di.Desired += rs.Desired
		di.Current += rs.Current
	}

	if di != nil && len(deployedClusters) == len(o.Clusters) {
		di.Deployed = true
	}
	return di
}
