package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	waypointPDBName       = "waypoint"
	waypointGatewayLabel = "gateway.networking.k8s.io/gateway-name" // Istio sets this on waypoint Deployment pods
	waypointSelectorVal  = "waypoint"
)

// EnsureWaypointPDB creates a PDB for the waypoint Deployment so Karpenter (or other disruptors)
// cannot evict the last waypoint pod (e.g. "Underutilized" consolidation). minAvailable: 1.
// The waypoint Deployment is created by Istio from the Gateway; we only create the PDB.
type EnsureWaypointPDB struct {
	Namespace string
}

func (p *EnsureWaypointPDB) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	minAvailable := intstr.FromInt(1)
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      waypointPDBName,
			Namespace: p.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{waypointGatewayLabel: waypointSelectorVal},
			},
		},
	}
	return plan.CreateOrUpdate(ctx, log, cli, pdb)
}

// DeleteWaypointPDB removes the waypoint PDB when switching off ambient / removing the waypoint.
type DeleteWaypointPDB struct {
	Namespace string
}

func (p *DeleteWaypointPDB) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      waypointPDBName,
			Namespace: p.Namespace,
		},
	}
	return utils.DeleteIfExists(ctx, cli, pdb)
}
