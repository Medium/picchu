package revision

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/prometheus/common/model"

	"github.com/golang/mock/gomock"
	"go.medium.engineering/picchu/pkg/prometheus"
	"go.medium.engineering/picchu/pkg/prometheus/mocks"

	testify "github.com/stretchr/testify/assert"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileRevision_Reconcile(t *testing.T) {
	log := test.MustNewLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(2)*time.Minute)
	defer cancel()
	scheme := runtime.NewScheme()
	picchu.AddToScheme(scheme)
	picchu.RegisterDefaults(scheme)
	ctrl := gomock.NewController(t)
	m := mocks.NewMockPromAPI(ctrl)
	m.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(model.Vector{}, nil, nil).AnyTimes()

	for _, test := range []struct {
		state    string
		expected bool
	}{
		{
			state:    "deployed",
			expected: false,
		},
		{
			state:    "deploying",
			expected: false,
		},
		{
			state:    "releasing",
			expected: false,
		},
		{
			state:    "released",
			expected: false,
		},
		{
			state:    "timingout",
			expected: true,
		},
	} {
		t.Run(test.state, func(t *testing.T) {
			assert := testify.New(t)
			fixtures := []client.Object{
				&picchu.Revision{
					ObjectMeta: meta.ObjectMeta{
						Name:      "rev",
						Namespace: "picchu",
						Labels: map[string]string{
							"label": "value",
						},
					},
					Spec: picchu.RevisionSpec{
						App: picchu.RevisionApp{
							Name:  "app",
							Ref:   "ref",
							Tag:   "v1",
							Image: "image",
						},
						Targets: []picchu.RevisionTarget{
							{
								Name:  "target",
								Fleet: "fleet",
							},
						},
					},
				},
				&picchu.ReleaseManager{
					ObjectMeta: meta.ObjectMeta{
						Name:      "app-target",
						Namespace: "picchu",
						Labels: map[string]string{
							picchu.LabelTarget: "target",
							picchu.LabelFleet:  "fleet",
							picchu.LabelApp:    "app",
						},
					},
					Status: picchu.ReleaseManagerStatus{
						Revisions: []picchu.ReleaseManagerRevisionStatus{
							{
								Tag: "v1",
								State: picchu.ReleaseManagerRevisionStateStatus{
									Current: test.state,
									Target:  test.state,
								},
							},
						},
					},
				},
			}
			//cli := fake.NewFakeClientWithScheme(scheme, fixtures...)
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(fixtures...).Build()
			reconciler := ReconcileRevision{
				client:       cli,
				scheme:       scheme,
				config:       utils.Config{},
				promAPI:      prometheus.InjectAPI(m, time.Duration(1)*time.Second),
				customLogger: log,
			}
			key := client.ObjectKeyFromObject(fixtures[0])
			res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			assert.NoError(err)
			assert.NotNil(res)

			instance := &picchu.Revision{}
			assert.NoError(cli.Get(ctx, key, instance))
			assert.Equal(test.expected, instance.Spec.Failed)
		})
	}
}
