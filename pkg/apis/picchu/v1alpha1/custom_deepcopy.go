package v1alpha1

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	istio "istio.io/api/networking/v1alpha3"
	"math"
)

var (
	_ = proto.Marshal
	_ = fmt.Errorf
	_ = math.Inf
)

// Istio uses proto.Clone, and therefore doesn't implement recursive deepCopy. If we want to use a subtype of istio's
// nested apis, we'll need to wrap the structs and create our own deepcopy impl.
func (in *Istio) DeepCopy() *Istio {
	if in == nil {
		return nil
	}
	out := new(Istio)
	in.DeepCopyInto(out)
	return out
}

func (in *Istio) DeepCopyInto(out *Istio) {
	if in.TrafficPolicy != nil {
		p := proto.Clone(in.TrafficPolicy).(*istio.TrafficPolicy)
		out.TrafficPolicy = p
	}
}

func (in *HTTPPortFault) DeepCopy() *HTTPPortFault {
	if in == nil {
		return nil
	}
	out := new(HTTPPortFault)
	in.DeepCopyInto(out)
	return out
}

func (in *HTTPPortFault) DeepCopyInto(out *HTTPPortFault) {
	if in.PortSelector != nil {
		p := proto.Clone(in.PortSelector).(*istio.PortSelector)
		out.PortSelector = p
	}
	if in.HTTPFault != nil {
		p := proto.Clone(in.HTTPFault).(*istio.HTTPFaultInjection)
		out.HTTPFault = p
	}
}
