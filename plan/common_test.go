package plan

import (
	"context"
	"testing"
	"time"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	test2 "go.medium.engineering/kubernetes/pkg/test"
	coreAsserts "go.medium.engineering/kubernetes/pkg/test/core/v1"
	externalSecretAsserts "go.medium.engineering/kubernetes/pkg/test/external-secrets/externalsecrets/v1"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	picchuScheme "go.medium.engineering/picchu/client/scheme"
	"go.medium.engineering/picchu/test"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	comparator = test2.NewComparator(picchuScheme.Scheme)
)

func init() {
	core.AddToScheme(picchuScheme.Scheme)
	es.AddToScheme(picchuScheme.Scheme)
	coreAsserts.RegisterAsserts(comparator)
	externalSecretAsserts.RegisterAsserts(comparator)
}

func TestIgnore(t *testing.T) {
	log := test.MustNewLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	for _, test := range []struct {
		Name     string
		Existing client.Object
		Updated  client.Object
		Expected client.Object
	}{
		{
			Name: "Update",
			Existing: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Data: map[string][]byte{
					"name": []byte("robert"),
				},
				Type: "Opaque",
			},
			Updated: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
				Data: map[string][]byte{
					"name": []byte("bob"),
				},
				Type: "Opaque",
			},
			Expected: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Data: map[string][]byte{
					"name": []byte("bob"),
				},
				Type: "Opaque",
			},
		},
		{
			Name: "Ignore",
			Existing: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
					Labels: map[string]string{
						picchu.LabelIgnore: "",
					},
				},
				Data: map[string][]byte{
					"name": []byte("robert"),
				},
				Type: "Opaque",
			},
			Updated: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
				Data: map[string][]byte{
					"name": []byte("bob"),
				},
				Type: "Opaque",
			},
			Expected: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
					Labels: map[string]string{
						picchu.LabelIgnore: "",
					},
				},
				Data: map[string][]byte{
					"name": []byte("robert"),
				},
				Type: "Opaque",
			},
		},
		{
			Name: "UpdateExternalSecret",
			Existing: &es.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: es.ExternalSecretSpec{
					SecretStoreRef: es.SecretStoreRef{
						Name: "app-cluster-secretstore",
						Kind: "ClusterSecretStore",
					},
					Target: es.ExternalSecretTarget{
						Name: "test",
					},
					DataFrom: []es.ExternalSecretDataFromRemoteRef{
						{
							Extract: &es.ExternalSecretDataRemoteRef{Key: "test-single-secret"},
						},
					},
				},
			},
			Updated: &es.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: es.ExternalSecretSpec{
					SecretStoreRef: es.SecretStoreRef{
						Name: "app-cluster-secretstore",
						Kind: "ClusterSecretStore",
					},
					Target: es.ExternalSecretTarget{
						Name: "test",
					},
					DataFrom: []es.ExternalSecretDataFromRemoteRef{
						{
							Extract: &es.ExternalSecretDataRemoteRef{Key: "test-multiple-secrets"},
						},
					},
				},
			},
			Expected: &es.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: es.ExternalSecretSpec{
					SecretStoreRef: es.SecretStoreRef{
						Name: "app-cluster-secretstore",
						Kind: "ClusterSecretStore",
					},
					Target: es.ExternalSecretTarget{
						Name: "test",
					},
					DataFrom: []es.ExternalSecretDataFromRemoteRef{
						{
							Extract: &es.ExternalSecretDataRemoteRef{Key: "test-multiple-secrets"},
						},
					},
				},
			},
		},
		{
			Name: "IgnoreExternalSecret",
			Existing: &es.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
					Labels: map[string]string{
						picchu.LabelIgnore: "",
					},
				},
				Spec: es.ExternalSecretSpec{
					SecretStoreRef: es.SecretStoreRef{
						Name: "app-cluster-secretstore",
						Kind: "ClusterSecretStore",
					},
					Target: es.ExternalSecretTarget{
						Name: "test",
					},
					DataFrom: []es.ExternalSecretDataFromRemoteRef{
						{
							Extract: &es.ExternalSecretDataRemoteRef{Key: "test-single-secret"},
						},
					},
				},
			},
			Updated: &es.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: es.ExternalSecretSpec{
					SecretStoreRef: es.SecretStoreRef{
						Name: "app-cluster-secretstore",
						Kind: "ClusterSecretStore",
					},
					Target: es.ExternalSecretTarget{
						Name: "test",
					},
					DataFrom: []es.ExternalSecretDataFromRemoteRef{
						{
							Extract: &es.ExternalSecretDataRemoteRef{Key: "test-multiple-secrets"},
						},
					},
				},
			},
			Expected: &es.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
					Labels: map[string]string{
						picchu.LabelIgnore: "",
					},
				},
				Spec: es.ExternalSecretSpec{
					SecretStoreRef: es.SecretStoreRef{
						Name: "app-cluster-secretstore",
						Kind: "ClusterSecretStore",
					},
					Target: es.ExternalSecretTarget{
						Name: "test",
					},
					DataFrom: []es.ExternalSecretDataFromRemoteRef{
						{
							Extract: &es.ExternalSecretDataRemoteRef{Key: "test-single-secret"},
						},
					},
				},
			},
		},
		{
			Name: "UpdateNamepsace",
			Existing: &core.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			Updated: &core.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"just": "doit",
					},
				},
			},
			Expected: &core.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"just": "doit",
					},
				},
			},
		},
		{
			Name: "IgnoreNamepsace",
			Existing: &core.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						picchu.LabelIgnore: "",
					},
				},
			},
			Updated: &core.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"just": "doit",
					},
				},
			},
			Expected: &core.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						picchu.LabelIgnore: "",
					},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithScheme(picchuScheme.Scheme).WithObjects(test.Existing).Build()
			CreateOrUpdate(ctx, log, cli, test.Updated)
			comparator.AssertMatch(ctx, t, cli, test.Expected)
		})
	}
}
