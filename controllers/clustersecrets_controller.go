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

	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterSecretsReconciler reconciles a ClusterSecrets object
type ClusterSecretsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	config utils.Config
}

// +kubebuilder:rbac:groups=picchu.medium.engineering.picchu.medium.engineering,resources=clustersecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=picchu.medium.engineering.picchu.medium.engineering,resources=clustersecrets/status,verbs=get;update;patch

func (r *ClusterSecretsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := r.Log.WithValues("clustersecrets", req.NamespacedName)

	instance := &picchuv1alpha1.ClusterSecrets{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	r.Scheme.Default(instance)

	rreq := newReconcileRequest(r, instance, reqLogger)

	// This means the cluster is new and needs the finalizer added
	if !instance.IsDeleted() {
		if instance.IsFinalized() {
			instance.AddFinalizer()
			return ctrl.Result{Requeue: true}, r.Client.Update(ctx, instance)
		}
		return ctrl.Result{}, rreq.reconcile(ctx)
	}
	if !instance.IsFinalized() {
		return ctrl.Result{}, rreq.finalize(ctx)
	}

	return ctrl.Result{}, nil

	return ctrl.Result{}, nil
}

func (r *ClusterSecretsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&picchuv1alpha1.ClusterSecrets{}).
		Complete(r)
}
