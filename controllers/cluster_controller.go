/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	config utils.Config
}

// +kubebuilder:rbac:groups=picchu.medium.engineering.picchu.medium.engineering,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=picchu.medium.engineering.picchu.medium.engineering,resources=clusters/status,verbs=get;update;patch

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := r.Log.WithValues("Request.Namespace", req.NamespacedName, "Request.Name", req.Name)

	// your logic here
	instance := &picchuv1alpha1.Cluster{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	r.Scheme.Default(instance)

	if !instance.IsDeleted() {
		if instance.IsFinalized() {
			instance.AddFinalizer()
			return ctrl.Result{Requeue: true}, r.Client.Update(context.TODO(), instance)
		}
		if !instance.Spec.Enabled {
			reqLogger.Info("Disabled, no status")
			instance.Status = picchuv1alpha1.ClusterStatus{}
		} else {
			secret := &corev1.Secret{}
			selector := types.NamespacedName{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			}
			err = r.Client.Get(context.TODO(), selector, secret)
			if err != nil {
				secret = nil
			}
			if secret != nil {
				config, err := instance.Config(secret)
				if err != nil {
					reqLogger.Error(err, "Failed to create kube config")
					return ctrl.Result{}, err
				}
				ready := true
				version, err := ServerVersion(config)
				if err != nil {
					ready = false
				}

				reqLogger.Info("Setting status")
				instance.Status = picchuv1alpha1.ClusterStatus{
					Kubernetes: picchuv1alpha1.ClusterKubernetesStatus{
						Version: FormatVersion(version),
						Ready:   ready,
					},
				}
			}
		}
		err = utils.UpdateStatus(context.TODO(), r.Client, instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Cluster status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: r.config.RequeueAfter}, nil
	}
	if !instance.IsFinalized() {
		instance.Finalize()
		return ctrl.Result{}, r.Client.Update(context.TODO(), instance)
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&picchuv1alpha1.Cluster{}).
		Complete(r)
}

func FormatVersion(version *version.Info) string {
	if version != nil && version.Major != "" && version.Minor != "" {
		return fmt.Sprintf("%s.%s", version.Major, version.Minor)
	} else {
		return ""
	}
}

func ServerVersion(config *rest.Config) (*version.Info, error) {
	disco, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	return disco.ServerVersion()
}
