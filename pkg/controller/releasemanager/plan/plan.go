package plan

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"

	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete any WorkerPodAutoScaler for the deployed revision
func deleteWPA(ctx context.Context, cli client.Client, namespace, name string) error {
	wpa := &wpav1.WorkerPodAutoScaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	if err := cli.Delete(ctx, wpa); err != nil {
		switch err.(type) {
		case *meta.NoKindMatchError:
			break
		default:
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}
