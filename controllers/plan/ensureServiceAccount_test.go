package plan

import (
	"context"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	_ "go.medium.engineering/kubernetes/pkg/test/core/v1"
)

var (
	ServiceAccount = &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name: "ServiceAccount",
			Labels: map[string]string{
				picchu.LabelOwnerName: "",
				picchu.LabelOwnerType: "",
			},
		},
	}
)

func TestCreatesServiceAccount(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	scheme := runtime.NewScheme()
	assert.NoError(core.AddToScheme(scheme))
	cli := fakeClient()

	en := &EnsureServiceAccount{
		Name: "ServiceAccount",
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertMatch(ctx, t, cli, ServiceAccount)
}

func TestIgnoreServiceAccount(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	scheme := runtime.NewScheme()
	assert.NoError(core.AddToScheme(scheme))
	cli := fakeClient(ServiceAccount)

	en := &EnsureServiceAccount{
		Name: "ServiceAccount",
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertMatch(ctx, t, cli, ServiceAccount)
}

func TestUpdatesServiceAccount(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	copy := ServiceAccount.DeepCopy()
	copy.Labels[picchu.LabelOwnerName] = "bob"
	cli := fakeClient(copy)

	en := &EnsureServiceAccount{
		Name: "ServiceAccount",
	}
	assert.NoError(en.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertMatch(ctx, t, cli, ServiceAccount)
}
