package releasemanager

import (
	"context"
	"math"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterInfo records info needed to compute expected replicas
type ClusterInfo struct {
	Name          string
	Live          bool
	ScalingFactor float64
}

// ClusterInfoList is info for all relevent active clusters
type ClusterInfoList []ClusterInfo

// ClusterCount returns the number of clusters by liveness (live vs. standby)
func (c *ClusterInfoList) ClusterCount(liveness bool) int {
	cnt := 0
	for i := range *c {
		if (*c)[i].Live == liveness {
			cnt++
		}
	}
	return cnt
}

// ExpectedReplicaCount returns expected total replicas by liveness adjusted for scalingFactors
func (c *ClusterInfoList) ExpectedReplicaCount(liveness bool, count int) int {
	perCluster := math.Ceil(float64(count) / float64(c.ClusterCount(true)))
	total := 0
	for i := range *c {
		cluster := (*c)[i]
		if cluster.Live == liveness {
			total += int(math.Ceil(perCluster * cluster.ScalingFactor))
		}
	}
	return total
}

type IncarnationController struct {
	deliveryClient  client.Client
	deliveryApplier plan.Applier
	planApplier     plan.Applier
	log             logr.Logger
	releaseManager  *picchuv1alpha1.ReleaseManager
	clusterInfo     ClusterInfoList
}

func (i *IncarnationController) getLog() logr.Logger {
	return i.log
}

func (i *IncarnationController) getReleaseManager() *picchuv1alpha1.ReleaseManager {
	return i.releaseManager
}

func (i *IncarnationController) getSecrets(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	secretList := &corev1.SecretList{}
	err := i.deliveryClient.List(ctx, secretList, opts)
	if err != nil {
		return nil, err
	}
	list := []runtime.Object{}
	for i := range secretList.Items {
		item := secretList.Items[i]
		list = append(list, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Labels: item.Labels,
				Name:   item.Name,
			},
			Type: item.Type,
			Data: item.Data,
		})
	}
	return list, err
}

func (i *IncarnationController) getConfigMaps(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	configMapList := &corev1.ConfigMapList{}
	err := i.deliveryClient.List(ctx, configMapList, opts)
	if err != nil {
		return nil, err
	}
	list := []runtime.Object{}
	for i := range configMapList.Items {
		item := configMapList.Items[i]
		list = append(list, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels: item.Labels,
				Name:   item.Name,
			},
			Data: item.Data,
		})
	}
	return list, err
}

func (i *IncarnationController) applyPlan(ctx context.Context, name string, p plan.Plan) error {
	return i.planApplier.Apply(ctx, p)
}

func (i *IncarnationController) applyDeliveryPlan(ctx context.Context, name string, p plan.Plan) error {
	return i.deliveryApplier.Apply(ctx, p)
}

func (i *IncarnationController) divideReplicas(count int32, percent int32) int32 {
	denominator := utils.Max(int32(i.clusterInfo.ClusterCount(true)), 1)
	factor := float64(utils.Min(100, percent)) / float64(100)
	answer := int32(math.Ceil(factor * (float64(count) / float64(denominator))))
	return utils.Max(answer, 1)
}

// Returns expected total live replicas accounting for scaling factors
func (i *IncarnationController) expectedTotalReplicas(count int32, percent int32) int32 {
	factor := float64(utils.Min(100, percent)) / float64(100)
	answer := int(math.Ceil(factor * float64(count)))
	return int32(i.clusterInfo.ExpectedReplicaCount(true, answer))
}

func (i *IncarnationController) liveCount() int {
	return i.clusterInfo.ClusterCount(true)
}
