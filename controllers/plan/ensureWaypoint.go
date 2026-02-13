package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	waypointGatewayName        = "waypoint"
	waypointGatewayClass       = "istio-waypoint"
	waypointLabelKey           = "istio.io/waypoint-for"
	waypointLabelValue         = "all"
	gatewayAPIGroup            = "gateway.networking.k8s.io"
	gatewayAPIVersion          = "v1"
	gatewayListenerPort  int64 = 15008
	gatewayListenerProto       = "HBONE"
)

var gatewayGVK = schema.GroupVersionKind{
	Group:   gatewayAPIGroup,
	Version: gatewayAPIVersion,
	Kind:    "Gateway",
}

// EnsureWaypoint creates or updates the Istio ambient waypoint Gateway in the namespace.
// Call this when AmbientMesh is true for the target.
type EnsureWaypoint struct {
	Namespace string
}

func (p *EnsureWaypoint) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gatewayGVK)
	u.SetNamespace(p.Namespace)
	u.SetName(waypointGatewayName)
	u.SetLabels(map[string]string{waypointLabelKey: waypointLabelValue})

	spec := map[string]interface{}{
		"gatewayClassName": waypointGatewayClass,
		"listeners": []interface{}{
			map[string]interface{}{
				"name":     "mesh",
				"port":     gatewayListenerPort,
				"protocol": gatewayListenerProto,
			},
		},
	}
	if err := unstructured.SetNestedMap(u.Object, spec, "spec"); err != nil {
		return err
	}

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(gatewayGVK)
	err := cli.Get(ctx, client.ObjectKey{Namespace: p.Namespace, Name: waypointGatewayName}, existing)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if createErr := cli.Create(ctx, u); createErr != nil {
			return createErr
		}
		log.Info("Created waypoint Gateway", "namespace", p.Namespace)
		return nil
	}

	// Only update if spec or labels actually differ (avoids no-op updates and log spam on every reconcile).
	existingSpec, _, _ := unstructured.NestedMap(existing.Object, "spec")
	existingLabels := existing.GetLabels()
	labelsMatch := existingLabels != nil && existingLabels[waypointLabelKey] == waypointLabelValue
	specMatch := specMapsEqual(existingSpec, spec)
	if labelsMatch && specMatch {
		return nil
	}

	existing.Object["spec"] = spec
	existing.SetLabels(map[string]string{waypointLabelKey: waypointLabelValue})
	if updateErr := cli.Update(ctx, existing); updateErr != nil {
		return updateErr
	}
	log.Info("Updated waypoint Gateway", "namespace", p.Namespace)
	return nil
}

// specMapsEqual compares gateway spec maps (handles listeners as []interface{}).
func specMapsEqual(a, b map[string]interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a["gatewayClassName"] != b["gatewayClassName"] {
		return false
	}
	al, okA := a["listeners"].([]interface{})
	bl, okB := b["listeners"].([]interface{})
	if !okA || !okB || len(al) != len(bl) {
		return okA == okB && (al == nil) == (bl == nil) && len(al) == len(bl)
	}
	for i := range al {
		am, _ := al[i].(map[string]interface{})
		bm, _ := bl[i].(map[string]interface{})
		if am["name"] != bm["name"] || am["protocol"] != bm["protocol"] {
			return false
		}
		// Port can be int64 or float64 (JSON unmarshaling) from the server.
		if !portEqual(am["port"], bm["port"]) {
			return false
		}
	}
	return true
}

func portEqual(a, b interface{}) bool {
	switch va := a.(type) {
	case int64:
		switch vb := b.(type) {
		case int64:
			return va == vb
		case float64:
			return float64(va) == vb
		}
	case float64:
		switch vb := b.(type) {
		case int64:
			return va == float64(vb)
		case float64:
			return va == vb
		}
	}
	return a == b
}

// DeleteWaypoint removes the Istio ambient waypoint Gateway from the namespace.
// Call this when switching from AmbientMesh to sidecar.
type DeleteWaypoint struct {
	Namespace string
}

func (p *DeleteWaypoint) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gatewayGVK)
	u.SetNamespace(p.Namespace)
	u.SetName(waypointGatewayName)

	err := cli.Delete(ctx, u)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err == nil {
		log.Info("Deleted waypoint Gateway", "namespace", p.Namespace)
	}
	return nil
}
