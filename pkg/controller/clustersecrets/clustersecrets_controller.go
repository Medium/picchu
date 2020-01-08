package clustersecrets

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("controller_clustersecrets")

// Add creates a new ClusterSecrets Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c utils.Config) reconcile.Reconciler {
	cfg, err := config.GetConfig()
	if err != nil {
		panic("Couldn't create client")
	}
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		panic("Couldn't create client")
	}
	return &ReconcileClusterSecrets{client: cli, scheme: mgr.GetScheme(), config: c}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	_, err := builder.ControllerManagedBy(mgr).
		ForType(&picchuv1alpha1.ClusterSecrets{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		Build(r)

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileClusterSecrets{}

// ReconcileClusterSecrets reconciles a ClusterSecrets object
type ReconcileClusterSecrets struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config utils.Config
}

// Reconcile reads that state of the cluster for a ClusterSecrets object and makes changes based on the state read
// and what is in the ClusterSecrets.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileClusterSecrets) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ClusterSecrets")
	ctx := context.TODO()

	// Fetch the ClusterSecrets instance
	instance := &picchuv1alpha1.ClusterSecrets{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
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
	r.scheme.Default(instance)

	rreq := newReconcileRequest(r, instance, reqLogger)

	// This means the cluster is new and needs the finalizer added
	if !instance.IsDeleted() {
		if instance.IsFinalized() {
			instance.AddFinalizer()
			return reconcile.Result{Requeue: true}, r.client.Update(ctx, instance)
		}
		return reconcile.Result{}, rreq.reconcile(ctx)
	}
	if !instance.IsFinalized() {
		return reconcile.Result{}, rreq.finalize(ctx)
	}

	return reconcile.Result{}, nil
}
