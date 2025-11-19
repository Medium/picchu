package plan

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EnsureRBAC struct {
	Name               string
	OwnerName          string
	OwnerType          string
	Namespace          string
	ServiceAccountName string
}

func (p *EnsureRBAC) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	// Create Role for external metrics access
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-external-metrics", p.Name),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelOwnerType: p.OwnerType,
				picchuv1alpha1.LabelOwnerName: p.OwnerName,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"external.metrics.k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	// Create or update Role
	if err := plan.CreateOrUpdate(ctx, log, cli, role); err != nil {
		return fmt.Errorf("failed to create/update Role: %w", err)
	}

	// Create RoleBinding
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-external-metrics-binding", p.Name),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelOwnerType: p.OwnerType,
				picchuv1alpha1.LabelOwnerName: p.OwnerName,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      p.ServiceAccountName,
				Namespace: p.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}

	// Create or update RoleBinding
	if err := plan.CreateOrUpdate(ctx, log, cli, roleBinding); err != nil {
		return fmt.Errorf("failed to create/update RoleBinding: %w", err)
	}

	return nil
}
