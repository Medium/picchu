package revision

import picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

type portInfo struct {
	picchu.PortInfo
	index int
}

func bucketIngressPorts(target picchu.RevisionTarget) map[picchu.PortMode][]portInfo {
	// bucket ports by mode
	track := map[picchu.PortMode][]portInfo{}
	// if there's multiple ports and no default is specified, the 'http' port will be defaulted
	for j := range target.Ports {
		port := target.Ports[j]
		track[port.Mode] = append(track[port.Mode], portInfo{
			PortInfo: port,
			index:    j,
		})
	}
	return track
}
