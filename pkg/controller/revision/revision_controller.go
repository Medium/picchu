package revision

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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
	_, err := builder.SimpleController().
		WithManager(mgr).
		ForType(&picchuv1alpha1.Revision{}).
		Owns(&picchuv1alpha1.Incarnation{}).
		Owns(&picchuv1alpha1.ReleaseManager{}).
		Build(r)

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

	statusBuilder := NewStatusBuilder(incarnations)

	// Set Revision instance as the owner and controller
	for _, incarnation := range incarnations {
		statusBuilder.UpdateStatus(incarnation, "creating")
		if err = controllerutil.SetControllerReference(instance, incarnation, r.scheme); err != nil {
			reqLogger.Error(err, "Failed to SetControllerReference for incarnation", "Incarnation.Name", incarnation.Name)
			break
		}
		_, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, incarnation, func(existing runtime.Object) error {
			statusBuilder.UpdateStatus(incarnation, "created")
			return nil
		})
		if err != nil {
			reqLogger.Error(err, "Failed to CreateOrUpdate incarnation", "Incarnation.Name", incarnation.Name)
			statusBuilder.UpdateStatus(incarnation, "failed")
			break
		}
	}

	instance.Status = *statusBuilder.Status()
	reqLogger.Info("Updating Revision status")
	if e := r.client.Status().Update(context.TODO(), instance); e != nil {
		reqLogger.Error(e, "Failed to update Revision status")
	}

	// Sync releasemanagers
	appName := instance.Spec.App.Name
	for _, target := range instance.Spec.Targets {
		targetName := target.Name
		clusters, err := r.getClustersByFleet(target.Fleet)
		if err != nil {
			return reconcile.Result{}, err
		}
		for _, cluster := range clusters.Items {
			clusterName := cluster.Name
			rmName := fmt.Sprintf("%s-%s-%s", appName, targetName, clusterName)
			rm := &picchuv1alpha1.ReleaseManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmName,
					Namespace: instance.Namespace,
				},
			}
			if err := controllerutil.SetControllerReference(instance, rm, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, rm, func(runtime.Object) error {
				rm.Spec.Cluster = clusterName
				rm.Spec.App = appName
				rm.Spec.Target = targetName
				return nil
			})
			if err != nil {
				return reconcile.Result{}, err
			}
			reqLogger.Info("ReleaseManager sync'd", "Op", op)
		}
	}

	return reconcile.Result{}, err
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
			ref := revision.Spec.App.Ref
			image := revision.Spec.App.Image
			commit := revision.Labels["medium.build/commit"]

			annotations := map[string]string{
				"git-scm.com/ref":                 ref,
				"git-scm.com/committer-timestamp": revision.Annotations["git-scm.com/committer-timestamp"],
				"github.com/repository":           revision.Annotations["github.com/repository"],
			}

			labels := map[string]string{
				"medium.build/target":   target.Name,
				"medium.build/fleet":    target.Fleet,
				"medium.build/cluster":  cluster.Name,
				"medium.build/revision": revision.Name,
				"medium.build/tag":      tag,
				"medium.build/app":      app,
				"medium.build/commit":   commit,
			}

			name := fmt.Sprintf("%s-%s-%s",
				tag,
				target.Name,
				cluster.Name,
			)

			incarnation := &picchuv1alpha1.Incarnation{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   revision.Namespace,
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: picchuv1alpha1.IncarnationSpec{
					App: picchuv1alpha1.IncarnationApp{
						Name:  app,
						Tag:   tag,
						Ref:   ref,
						Image: image,
					},
					Assignment: picchuv1alpha1.IncarnationAssignment{
						Name:   cluster.ObjectMeta.Name,
						Target: target.Name,
					},
					Scale:   target.Scale,
					Release: target.Release,
					Ports:   revision.Spec.Ports,
				},
			}
			incarnation.Spec.Release.Eligible = revision.Spec.Release.Eligible
			incarnations = append(incarnations, incarnation)
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

type StatusBuilder struct {
	TargetStatuses map[string]picchuv1alpha1.RevisionTargetStatus
}

func NewStatusBuilder(incarnations []*picchuv1alpha1.Incarnation) *StatusBuilder {
	targetStatuses := map[string]picchuv1alpha1.RevisionTargetStatus{}
	for _, incarnation := range incarnations {
		target := incarnation.Spec.Assignment.Target
		targetStatus, ok := targetStatuses[target]
		if !ok {
			targetStatus = picchuv1alpha1.RevisionTargetStatus{
				Name:         target,
				Incarnations: []picchuv1alpha1.RevisionTargetIncarnationStatus{},
			}
		}
		targetStatus.Incarnations = append(targetStatus.Incarnations, picchuv1alpha1.RevisionTargetIncarnationStatus{
			Name:    incarnation.Name,
			Cluster: incarnation.Spec.Assignment.Name,
			Status:  "unknown",
		})
		targetStatuses[target] = targetStatus
	}
	return &StatusBuilder{targetStatuses}
}

func (s *StatusBuilder) UpdateStatus(incarnation *picchuv1alpha1.Incarnation, status string) {
	targetStatus := s.TargetStatuses[incarnation.Spec.Assignment.Target]
	for i, incStatus := range targetStatus.Incarnations {
		if incStatus.Name == incarnation.Name {
			targetStatus.Incarnations[i].Status = status
			s.TargetStatuses[targetStatus.Name] = targetStatus
			return
		}
	}
	return
}

func (s *StatusBuilder) Status() *picchuv1alpha1.RevisionStatus {
	targets := make([]picchuv1alpha1.RevisionTargetStatus, len(s.TargetStatuses))
	var i int
	for _, v := range s.TargetStatuses {
		targets[i] = v
		i++
	}
	return &picchuv1alpha1.RevisionStatus{targets}
}
