package cluster

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var log = logf.Log.WithName("controller_cluster")

// Add creates a new Cluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Cluster
	err = c.Watch(&source.Kind{Type: &picchuv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCluster{}

// ReconcileCluster reconciles a Cluster object
type ReconcileCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cluster")

	// Fetch the Cluster instance
	instance := &picchuv1alpha1.Cluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if !instance.Spec.Enabled {
		reqLogger.Info("Disabled, no status")
		instance.Status = picchuv1alpha1.ClusterStatus{}
	} else {
		// TODO(bob): get status of cluster with API call
		secret := &corev1.Secret{}
		selector := types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}
		err = r.client.Get(context.TODO(), selector, secret)
		if err != nil {
			secret = nil
		} else {
			for k, _ := range secret.Data {
				reqLogger.Info(fmt.Sprintf("Secret data %s", k))
			}
		}
		config, err := instance.Config(secret)
		if err != nil {
			reqLogger.Error(err, "Failed to create kube config")
			return reconcile.Result{}, err
		}
		ready := "True"
		version, err := ServerVersion(config)
		if err != nil {
			ready = "False"
		}

		reqLogger.Info("Setting status")
		instance.Status = picchuv1alpha1.ClusterStatus{
			Kubernetes: picchuv1alpha1.ClusterKubernetesStatus{
				Version: FormatVersion(version),
			},
			Conditions: []picchuv1alpha1.ClusterConditionStatus{
				picchuv1alpha1.ClusterConditionStatus{"Ready", ready},
			},
		}
	}
	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to update Cluster status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
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
