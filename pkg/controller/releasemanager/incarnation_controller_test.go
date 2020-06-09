package releasemanager

import (
	"context"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"
	istio "istio.io/api/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	T "testing"
	"time"

	"go.medium.engineering/picchu/pkg/test"

	testify "github.com/stretchr/testify/assert"
)

func TestDivideReplicas(t *T.T) {
	assert := testify.New(t)
	log := test.MustNewLogger()
	out := IncarnationController{
		clusterInfo: ClusterInfoList{{
			Name:          "cluster-0",
			Live:          true,
			ScalingFactor: 1.0,
		}, {
			Name:          "cluster-1",
			Live:          true,
			ScalingFactor: 1.0,
		}, {
			Name:          "cluster-2",
			Live:          true,
			ScalingFactor: 1.0,
		}, {
			Name:          "cluster-3",
			Live:          true,
			ScalingFactor: 1.0,
		}},
		log: log,
	}
	// 5 are required, so each cluster should get 2 for a total of 8
	// since 1 would be 4 and 4 < 5
	assert.Equal(int32(2), out.divideReplicas(5, 100))
	assert.Equal(int32(1), out.divideReplicas(5, 80))
	assert.Equal(int32(2), out.divideReplicas(5, 81))
	assert.Equal(int32(8), out.expectedTotalReplicas(5, 81))
	assert.Equal(int32(8), out.expectedTotalReplicas(5, 81))
	assert.Equal(int32(8), out.expectedTotalReplicas(5, 81))
}

func TestDivideReplicasWithScaling(t *T.T) {
	assert := testify.New(t)
	log := test.MustNewLogger()
	out := IncarnationController{
		clusterInfo: ClusterInfoList{{
			Name:          "cluster-0",
			Live:          true,
			ScalingFactor: 2.0,
		}, {
			Name:          "cluster-1",
			Live:          true,
			ScalingFactor: 2.0,
		}, {
			Name:          "cluster-2",
			Live:          true,
			ScalingFactor: 3.0,
		}, {
			Name:          "cluster-3",
			Live:          true,
			ScalingFactor: 1.0,
		}},
		log: log,
	}
	// 5 are required, so each cluster should get 2 for a total of 8
	// since 1 would be 4 and 4 < 5
	assert.EqualValues(2, out.divideReplicas(5, 100))
	assert.Equal(int32(1), out.divideReplicas(5, 80))
	assert.Equal(int32(2), out.divideReplicas(5, 81))
	assert.Equal(int32(16), out.expectedTotalReplicas(5, 81))
	assert.Equal(int32(16), out.expectedTotalReplicas(5, 81))
	assert.Equal(int32(16), out.expectedTotalReplicas(5, 81))
}

func TestGetFaults(t *T.T) {
	assert := testify.New(t)
	log := test.MustNewLogger()
	fixture := &picchuv1alpha1.FaultInjector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-fault",
			Namespace: "picchu",
			Labels: map[string]string{
				picchuv1alpha1.LabelApp:    "echo",
				picchuv1alpha1.LabelTarget: "delivery",
			},
		},
		Spec: picchuv1alpha1.FaultInjectorSpec{HTTPPortFaults: []picchuv1alpha1.HTTPPortFault{
			{
				PortSelector: &istio.PortSelector{Number: 80},
				HTTPFault: &istio.HTTPFaultInjection{
					Abort: &istio.HTTPFaultInjection_Abort{
						ErrorType: &istio.HTTPFaultInjection_Abort_HttpStatus{
							HttpStatus: 404,
						},
						Percentage: &istio.Percent{
							Value: 100,
						},
					},
				},
			},
		}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Second)
	defer cancel()
	scheme := runtime.NewScheme()
	picchuv1alpha1.AddToScheme(scheme)
	cli := fake.NewFakeClientWithScheme(scheme, fixture)
	controller := ReconcileReleaseManager{
		client: cli,
		scheme: scheme,
		config: utils.Config{},
	}
	faults, err := controller.getFaults(ctx, log, "picchu", "echo", "delivery")
	assert.NoError(err)
	assert.Equal(1, len(faults))

	faults, err = controller.getFaults(ctx, log, "picchu", "xecho", "delivery")
	assert.NoError(err)
	assert.Equal(0, len(faults))

	faults, err = controller.getFaults(ctx, log, "picchu", "echo", "xdelivery")
	assert.NoError(err)
	assert.Equal(0, len(faults))

	faults, err = controller.getFaults(ctx, log, "picchu", "xecho", "xdelivery")
	assert.NoError(err)
	assert.Equal(0, len(faults))

	faults, err = controller.getFaults(ctx, log, "xpicchu", "echo", "delivery")
	assert.NoError(err)
	assert.Equal(0, len(faults))
}
