package revision

import picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

type portInfo struct {
	picchu.PortInfo
	index int
}

// returns a map of ingresses and ports mapped to ingresses
func bucketIngressPorts(target picchu.RevisionTarget) map[string][]portInfo {
	// bucket ports by mode
	track := map[string][]portInfo{}
	for i := range target.Ports {
		pi := portInfo{
			PortInfo: target.Ports[i],
			index:    i,
		}
		if pi.Ingresses != nil {
			for _, ingress := range pi.Ingresses {
				track[ingress] = append(track[ingress], pi)
			}
			continue
		}

		switch pi.Mode {
		case picchu.PortPublic:
			track["public"] = append(track["public"], pi)
			track["private"] = append(track["private"], pi)
		case picchu.PortPrivate:
			track["private"] = append(track["private"], pi)
		}
	}
	return track
}
