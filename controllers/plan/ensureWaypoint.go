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

	existing.Object["spec"] = spec
	existing.SetLabels(map[string]string{waypointLabelKey: waypointLabelValue})
	if updateErr := cli.Update(ctx, existing); updateErr != nil {
		return updateErr
	}
	log.Info("Updated waypoint Gateway", "namespace", p.Namespace)
	return nil
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
