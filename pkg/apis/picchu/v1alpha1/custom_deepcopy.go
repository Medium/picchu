package v1alpha1

import (
	"fmt"
	proto "github.com/gogo/protobuf/proto"
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
// api, we'll need a deepcopy impl.
func (in *Istio) DeepCopy() *Istio {
	if in == nil {
		return nil
	}
	out := new(Istio)
	in.DeepCopyInto(out)
	return out
}

func (in *Istio) DeepCopyInto(out *Istio) {
	p := proto.Clone(in.TrafficPolicy).(*istio.TrafficPolicy)
	*out.TrafficPolicy = *p
}
