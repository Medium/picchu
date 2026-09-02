package plan

import (
	"context"
	"testing"
	"time"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/stretchr/testify/assert"
	test2 "go.medium.engineering/kubernetes/pkg/test"
	coreAsserts "go.medium.engineering/kubernetes/pkg/test/core/v1"
	externalSecretAsserts "go.medium.engineering/kubernetes/pkg/test/external-secrets/externalsecrets/v1"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	picchuScheme "go.medium.engineering/picchu/client/scheme"
	"go.medium.engineering/picchu/test"
	appsv1 "k8s.io/api/apps/v1"
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
	appsv1.AddToScheme(picchuScheme.Scheme)
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

func TestCreateOrUpdateReplicaSetKarpenterDoNotDisrupt(t *testing.T) {
	log := test.MustNewLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	const (
		namespace = "testnamespace"
		name      = "testtag"
	)

	podLabels := map[string]string{
		"app":                           "testapp",
		"tag.picchu.medium.engineering": name,
	}
	podAnnotations := func(karpenterValue string) map[string]string {
		annotations := map[string]string{
			"other-annotation": "keep-me",
		}
		if karpenterValue != "" {
			annotations[picchu.AnnotationKarpenterDoNotDisrupt] = karpenterValue
		}
		return annotations
	}
	replicas := int32(1)
	replicaSet := func(image string, karpenterValue string) *appsv1.ReplicaSet {
		return &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: &replicas,
				Selector: metav1.SetAsLabelSelector(podLabels),
				Template: core.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      podLabels,
						Annotations: podAnnotations(karpenterValue),
					},
					Spec: core.PodSpec{
						Containers: []core.Container{{
							Name:  "app",
							Image: image,
						}},
					},
				},
			},
		}
	}

	for _, test := range []struct {
		Name             string
		Existing         *appsv1.ReplicaSet
		Updated          *appsv1.ReplicaSet
		ExpectedKarpenter string
		ExpectedImage    string
	}{
		{
			Name:              "removes annotation after deploying",
			Existing:          replicaSet("old-image:tag", "30m"),
			Updated:           replicaSet("new-image:tag", ""),
			ExpectedKarpenter: "",
			ExpectedImage:     "old-image:tag",
		},
		{
			Name:              "adds annotation while deploying",
			Existing:          replicaSet("old-image:tag", ""),
			Updated:           replicaSet("new-image:tag", "30m"),
			ExpectedKarpenter: "30m",
			ExpectedImage:     "old-image:tag",
		},
		{
			Name:              "updates annotation while deploying",
			Existing:          replicaSet("old-image:tag", "30m"),
			Updated:           replicaSet("new-image:tag", "true"),
			ExpectedKarpenter: "true",
			ExpectedImage:     "old-image:tag",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithScheme(picchuScheme.Scheme).WithObjects(test.Existing).Build()
			assert.NoError(t, CreateOrUpdate(ctx, log, cli, test.Updated))

			got := &appsv1.ReplicaSet{}
			assert.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, got))
			assert.Equal(t, test.ExpectedImage, got.Spec.Template.Spec.Containers[0].Image)
			assert.Equal(t, "keep-me", got.Spec.Template.Annotations["other-annotation"])
			if test.ExpectedKarpenter == "" {
				_, ok := got.Spec.Template.Annotations[picchu.AnnotationKarpenterDoNotDisrupt]
				assert.False(t, ok)
			} else {
				assert.Equal(t, test.ExpectedKarpenter, got.Spec.Template.Annotations[picchu.AnnotationKarpenterDoNotDisrupt])
			}
		})
	}
}
