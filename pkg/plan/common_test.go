package plan

import (
	"context"
	test2 "go.medium.engineering/kubernetes/pkg/test"
	coreAsserts "go.medium.engineering/kubernetes/pkg/test/core/v1"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	picchuScheme "go.medium.engineering/picchu/pkg/client/scheme"
	"go.medium.engineering/picchu/pkg/test"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

var (
	comparator = test2.NewComparator(picchuScheme.Scheme)
)

func init() {
	core.AddToScheme(picchuScheme.Scheme)
	coreAsserts.RegisterAsserts(comparator)
}

func TestIgnore(t *testing.T) {
	log := test.MustNewLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	for _, test := range []struct {
		Name     string
		Existing runtime.Object
		Updated  runtime.Object
		Expected runtime.Object
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
			cli := fake.NewFakeClientWithScheme(picchuScheme.Scheme, test.Existing)
			CreateOrUpdate(ctx, log, cli, test.Updated)
			comparator.AssertMatch(ctx, t, cli, test.Expected)
		})
	}
}
