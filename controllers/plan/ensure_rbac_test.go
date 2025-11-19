package plan

import (
	"context"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
)

var (
	testCluster = &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
	}

	testRole = &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp-external-metrics",
			Namespace: "testns",
			Labels: map[string]string{
				picchuv1alpha1.LabelOwnerName: "test-owner",
				picchuv1alpha1.LabelOwnerType: "ReleaseManager",
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

	testRoleBinding = &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp-external-metrics-binding",
			Namespace: "testns",
			Labels: map[string]string{
				picchuv1alpha1.LabelOwnerName: "test-owner",
				picchuv1alpha1.LabelOwnerType: "ReleaseManager",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "test-sa",
				Namespace: "testns",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "testapp-external-metrics",
		},
	}
)

func TestEnsureRBAC_CreatesResources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert := assert.New(t)
	log := test.MustNewLogger()

	// Create a fake client with the test scheme
	cli := fakeClient()

	en := &EnsureRBAC{
		Name:               "testapp",
		OwnerName:          "test-owner",
		OwnerType:          "ReleaseManager",
		Namespace:          "testns",
		ServiceAccountName: "test-sa",
	}

	err := en.Apply(ctx, cli, testCluster, log)
	assert.NoError(err, "Should not return error")

	// Verify Role was created
	var role rbacv1.Role
	err = cli.Get(ctx, client.ObjectKey{Name: "testapp-external-metrics", Namespace: "testns"}, &role)
	assert.NoError(err)
	assert.Equal("testapp-external-metrics", role.Name)

	// Verify RoleBinding was created
	var binding rbacv1.RoleBinding
	err = cli.Get(ctx, client.ObjectKey{Name: "testapp-external-metrics-binding", Namespace: "testns"}, &binding)
	assert.NoError(err)
	assert.Equal("testapp-external-metrics-binding", binding.Name)
}

func TestEnsureRBAC_UpdatesExisting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert := assert.New(t)
	log := test.MustNewLogger()

	// Create existing role with different rules
	existingRole := testRole.DeepCopy()
	existingRole.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"some.other.api"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
	}

	cli := fakeClient(existingRole, testRoleBinding)

	en := &EnsureRBAC{
		Name:               "testapp",
		OwnerName:          "test-owner",
		OwnerType:          "ReleaseManager",
		Namespace:          "testns",
		ServiceAccountName: "test-sa",
	}

	err := en.Apply(ctx, cli, testCluster, log)
	assert.NoError(err)

	// Verify Role was updated
	var role rbacv1.Role
	err = cli.Get(ctx, client.ObjectKey{Name: "testapp-external-metrics", Namespace: "testns"}, &role)
	assert.NoError(err)
	assert.Equal("external.metrics.k8s.io", role.Rules[0].APIGroups[0])
}
