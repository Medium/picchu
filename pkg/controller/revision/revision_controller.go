package revision

import (
	"context"
	"fmt"
	"strings"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

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
func Add(mgr manager.Manager, c utils.Config) error {
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c utils.Config) reconcile.Reconciler {
	return &ReconcileRevision{client: mgr.GetClient(), scheme: mgr.GetScheme(), config: c}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	_, err := builder.SimpleController().
		WithManager(mgr).
		ForType(&picchuv1alpha1.Revision{}).
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
	config utils.Config
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
	r.scheme.Default(instance)
	if err = r.LabelWithAppAndFleets(instance); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.SyncReleaseManagersForRevision(instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}

func (r *ReconcileRevision) LabelWithAppAndFleets(revision *picchuv1alpha1.Revision) error {
	fleetLabels := []string{}
	updated := false
	for _, target := range revision.Spec.Targets {
		name := fmt.Sprintf("%s%s", picchuv1alpha1.LabelFleetPrefix, target.Fleet)
		fleetLabels = append(fleetLabels, name)
		if _, ok := revision.Labels[name]; !ok {
			revision.Labels[name] = ""
			updated = true
		}
	}
	for name, _ := range revision.Labels {
		if strings.HasPrefix(name, picchuv1alpha1.LabelFleetPrefix) {
			found := false
			for _, expected := range fleetLabels {
				if name == expected {
					found = true
					break
				}
			}
			if !found {
				delete(revision.Labels, name)
				updated = true
			}
		}
	}

	if _, ok := revision.Labels[picchuv1alpha1.LabelApp]; !ok {
		revision.Labels[picchuv1alpha1.LabelApp] = revision.Spec.App.Name
		updated = true
	}

	if updated {
		return r.client.Update(context.TODO(), revision)
	}
	return nil
}

func (r *ReconcileRevision) GetOrCreateReleaseManager(
	target *picchuv1alpha1.RevisionTarget,
	cluster *picchuv1alpha1.Cluster,
	revision *picchuv1alpha1.Revision,
) *picchuv1alpha1.ReleaseManager {
	labels := map[string]string{
		picchuv1alpha1.LabelTarget:  target.Name,
		picchuv1alpha1.LabelFleet:   target.Fleet,
		picchuv1alpha1.LabelCluster: cluster.Name,
		picchuv1alpha1.LabelApp:     revision.Spec.App.Name,
	}
	rms := &picchuv1alpha1.ReleaseManagerList{}
	opts := client.
		MatchingLabels(labels).
		InNamespace(revision.Namespace)
	r.client.List(context.TODO(), opts, rms)
	if len(rms.Items) > 1 {
		panic(fmt.Sprintf("Too many ReleaseManagers matching %#v", labels))
	}
	if len(rms.Items) == 1 {
		return &rms.Items[0]
	}
	return &picchuv1alpha1.ReleaseManager{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", revision.Spec.App.Name),
			Namespace:    revision.Namespace,
			Labels:       labels,
			Finalizers: []string{
				picchuv1alpha1.FinalizerReleaseManager,
			},
		},
	}
}

func (r *ReconcileRevision) SyncReleaseManagersForRevision(revision *picchuv1alpha1.Revision) error {
	// Sync releasemanagers
	appName := revision.Spec.App.Name
	expiredCount := 0
	rmCount := 0
	for _, target := range revision.Spec.Targets {
		targetName := target.Name
		clusters, err := r.getClustersByFleet(revision.Namespace, target.Fleet)
		if err != nil {
			return err
		}
		for _, cluster := range clusters.Items {
			if cluster.IsDeleted() {
				continue
			}
			rm := r.GetOrCreateReleaseManager(&target, &cluster, revision)
			op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, rm, func(runtime.Object) error {
				rm.Spec.Cluster = cluster.Name
				rm.Spec.App = appName
				rm.Spec.Target = targetName
				return nil
			})
			if err != nil {
				log.Error(err, "Failed to sync releaseManager")
				return err
			}

			rmCount++
			for _, rl := range rm.Status.Revisions {
				if rl.Tag == revision.Spec.App.Tag && rl.Expired == true {
					expiredCount++
				}
			}

			log.Info("ReleaseManager sync'd", "Op", op)
		}
		if len(clusters.Items) == 0 {
			log.Info("No clusters found in target fleet", "Target.Fleet", target.Fleet)
		}
	}

	// if Revision is expired in all ReleaseManagers, without deleting
	// clusterless revisions
	if expiredCount == rmCount && expiredCount > 0 {
		if err := r.DeleteRevision(revision); err != nil {
			log.Error(err, "Failed to delete Revision")
			return err
		}
	}
	return nil
}

func (r *ReconcileRevision) getClustersByFleet(namespace string, fleet string) (*picchuv1alpha1.ClusterList, error) {
	clusters := &picchuv1alpha1.ClusterList{}
	opts := client.
		MatchingLabels(map[string]string{picchuv1alpha1.LabelFleet: fleet}).
		InNamespace(namespace)
	err := r.client.List(context.TODO(), opts, clusters)
	r.scheme.Default(clusters)
	return clusters, err
}

func (r *ReconcileRevision) DeleteRevision(revision *picchuv1alpha1.Revision) error {
	log.Info("Deleting revision", "Name", revision.Name, "Namespace", revision.Namespace)
	if err := r.client.Delete(context.TODO(), revision); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}
