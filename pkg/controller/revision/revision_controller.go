package revision

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_revision")

// Add creates a new Revision Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRevision{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("revision-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Revision
	err = c.Watch(&source.Kind{Type: &picchuv1alpha1.Revision{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Incarnations and requeue the owner Revision
	err = c.Watch(&source.Kind{Type: &picchuv1alpha1.Incarnation{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &picchuv1alpha1.Revision{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRevision{}

// ReconcileRevision reconciles a Revision object
type ReconcileRevision struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Revision object and makes changes based on the state read
// and what is in the Revision.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRevision) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Revision")

	// Fetch the Revision instance
	instance := &picchuv1alpha1.Revision{}
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

	// Define new Incarnation objects for the Revision
	incarnations, err := r.newIncarnationsForRevision(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Set Revision instance as the owner and controller
	for _, incarnation := range incarnations {
		if err := controllerutil.SetControllerReference(instance, incarnation, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, incarnation, func(existing runtime.Object) error {
			i := existing.(*picchuv1alpha1.Incarnation)
			reqLogger.Info("Skip reconcile: Incarnation already exists", "Incarnation.Namespace", i.Namespace, "Incarnation.Name", i.Name)
			return nil
		})
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Incarnation reconciled", "Operation", op)
	}

	return reconcile.Result{}, nil
}

// newIncarnationsForRevision returns incarnations for all target clusters  for a Revision
func (r *ReconcileRevision) newIncarnationsForRevision(revision *picchuv1alpha1.Revision) ([]*picchuv1alpha1.Incarnation, error) {
	var incarnations []*picchuv1alpha1.Incarnation
	for _, target := range revision.Spec.Targets {
		clusters, err := r.getClustersByFleet(target.Fleet)
		if err != nil {
			return incarnations, err
		}
		for _, cluster := range clusters.Items {
			app := revision.Spec.App.Name
			tag := revision.Spec.App.Tag
			commit := revision.Labels["medium.build/commit"]

			labels := map[string]string{
				"medium.build/target":   target.Name,
				"medium.build/fleet":    target.Fleet,
				"medium.build/cluster":  cluster.Name,
				"medium.build/revision": revision.ObjectMeta.Name,
				"medium.build/tag":      tag,
				"medium.build/app":      app,
				"medium.build/commit":   commit,
			}

			name := fmt.Sprintf("%s-%s-%s-%s",
				app,
				target.Name,
				cluster.Name,
				commit[0:11],
			)

			incarnations = append(incarnations, &picchuv1alpha1.Incarnation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: revision.Namespace,
					Labels:    labels,
				},
				Spec: picchuv1alpha1.IncarnationSpec{
					App: picchuv1alpha1.IncarnationApp{
						Name: app,
						Tag:  tag,
					},
					Assignment: picchuv1alpha1.IncarnationAssignment{
						Name: cluster.ObjectMeta.Name,
					},
					Scale: picchuv1alpha1.IncarnationScale{
						Min: 1,
						Max: 10,
						Resources: []picchuv1alpha1.IncarnationScaleResource{
							picchuv1alpha1.IncarnationScaleResource{
								CPU: "200m",
							},
						},
					},
					Release: picchuv1alpha1.IncarnationRelease{
						Max:      100,
						Rate:     "TX",
						Schedule: "Humane",
					},
				},
			})
		}
	}
	return incarnations, nil
}

func (r *ReconcileRevision) getClustersByFleet(fleet string) (*picchuv1alpha1.ClusterList, error) {
	clusters := &picchuv1alpha1.ClusterList{}
	opts := client.MatchingLabels(map[string]string{"medium.build/fleet": fleet})
	err := r.client.List(context.TODO(), opts, clusters)
	return clusters, err
}
