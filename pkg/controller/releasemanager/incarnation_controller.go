package releasemanager

import (
	"context"
	"math"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IncarnationController struct {
	deliveryClient client.Client
	planApplier    plan.Applier
	log            logr.Logger
	releaseManager *picchuv1alpha1.ReleaseManager
	fleetSize      int32
}

func (i *IncarnationController) getLog() logr.Logger {
	return i.log
}

func (i *IncarnationController) getReleaseManager() *picchuv1alpha1.ReleaseManager {
	return i.releaseManager
}

func (i *IncarnationController) getSecrets(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	secretList := &corev1.SecretList{}
	err := i.deliveryClient.List(ctx, opts, secretList)
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
	err := i.deliveryClient.List(ctx, opts, configMapList)
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
	i.log.Info("Applying plan", "Name", name, "Plan", p)
	return i.planApplier.Apply(ctx, p)
}

func (i *IncarnationController) divideReplicas(count int32, percent int32) int32 {
	denominator := utils.Max(i.fleetSize, 1)
	percent = utils.Min(100, percent)
	answer := int32(math.Ceil((float64(percent) / float64(100)) * (float64(count) / float64(denominator))))
	return utils.Max(answer, 1)
}

func (i *IncarnationController) expectedTotalReplicas(count int32, percent int32) int32 {
	return i.divideReplicas(count, precent) * utils.Max(i.fleetSize, 1)
}
