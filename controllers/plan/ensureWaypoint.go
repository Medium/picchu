package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	waypointGatewayName            = "waypoint"
	waypointGatewayClass           = "istio-waypoint"
	waypointLabelKey               = "istio.io/waypoint-for"
	waypointLabelValue             = "all"
	waypointOptionsConfigMap       = "waypoint-options"
	gatewayAPIGroup                = "gateway.networking.k8s.io"
	gatewayAPIVersion              = "v1"
	gatewayListenerPort      int64 = 15008
	gatewayListenerProto           = "HBONE"
)

// waypointDeploymentOverlay is the Istio parametersRef "deployment" overlay: soft pod anti-affinity
// so waypoint pods prefer to run on different hosts (Karpenter can still consolidate when needed).
const waypointDeploymentOverlay = `spec:
  template:
    metadata:
      annotations:
        ad.datadoghq.com/istio-proxy.checks: '{"istio":{"init_config":{},"instances":[{"use_openmetrics":true,"istio_mode":"ambient","waypoint_endpoint":"http://%%host%%:15020/stats/prometheus","collect_histogram_buckets":true}]}}'
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  gateway.networking.k8s.io/gateway-name: waypoint
              topologyKey: kubernetes.io/hostname
`

var gatewayGVK = schema.GroupVersionKind{
	Group:   gatewayAPIGroup,
	Version: gatewayAPIVersion,
	Kind:    "Gateway",
}

// EnsureWaypointOptions creates or updates the ConfigMap referenced by the waypoint Gateway's
// spec.infrastructure.parametersRef. It contains the "deployment" overlay with soft pod anti-affinity
// so waypoint pods prefer different hosts.
type EnsureWaypointOptions struct {
	Namespace string
}

func (p *EnsureWaypointOptions) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      waypointOptionsConfigMap,
			Namespace: p.Namespace,
		},
		Data: map[string]string{
			"deployment": waypointDeploymentOverlay,
		},
	}
	return plan.CreateOrUpdate(ctx, log, cli, cm)
}

// DeleteWaypointOptions removes the waypoint options ConfigMap when the waypoint is removed.
type DeleteWaypointOptions struct {
	Namespace string
}

func (p *DeleteWaypointOptions) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      waypointOptionsConfigMap,
			Namespace: p.Namespace,
		},
	}
	err := cli.Delete(ctx, cm)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
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
		"infrastructure": map[string]interface{}{
			"parametersRef": map[string]interface{}{
				"group": "",
				"kind":  "ConfigMap",
				"name":  waypointOptionsConfigMap,
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

// specMapsEqual compares gateway spec maps (handles listeners and infrastructure).
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
	if !infrastructureEqual(a["infrastructure"], b["infrastructure"]) {
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

func infrastructureEqual(a, b interface{}) bool {
	am, okA := a.(map[string]interface{})
	bm, okB := b.(map[string]interface{})
	if !okA && !okB {
		return true
	}
	if !okA || !okB {
		return false
	}
	prA, _ := am["parametersRef"].(map[string]interface{})
	prB, _ := bm["parametersRef"].(map[string]interface{})
	if prA == nil && prB == nil {
		return true
	}
	if prA == nil || prB == nil {
		return false
	}
	return prA["group"] == prB["group"] && prA["kind"] == prB["kind"] && prA["name"] == prB["name"]
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
