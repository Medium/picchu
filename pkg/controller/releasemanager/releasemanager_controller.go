package releasemanager

import (
	"context"
	"fmt"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	"go.medium.engineering/picchu/pkg/controller/utils"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	clog                         = logf.Log.WithName("controller_releasemanager")
	incarnationGitReleaseLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "picchu_git_release_latency",
		Help:    "track time from git revision creation to incarnation release",
		Buckets: prometheus.ExponentialBuckets(10, 3, 7),
	})
	incarnationGitDeployLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "picchu_git_deploy_latency",
		Help:    "track time from git revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	})
	incarnationRevisionDeployLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "picchu_revision_deploy_latency",
		Help:    "track time from revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	})
	revisionReleaseWeightGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_revision_release_weight",
		Help: "Percent of traffic a revision is getting as a target release",
	}, []string{"app", "tag", "target", "target_cluster"})
	incarnationRevisionRollbackLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "picchu_revision_rollback_latency",
		Help:    "track time from failed revision to rollbacked incarnation",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	})
)

// Add creates a new ReleaseManager Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	metrics.Registry.MustRegister(incarnationGitReleaseLatency)
	metrics.Registry.MustRegister(incarnationGitDeployLatency)
	metrics.Registry.MustRegister(incarnationRevisionDeployLatency)
	metrics.Registry.MustRegister(incarnationRevisionRollbackLatency)
	metrics.Registry.MustRegister(revisionReleaseWeightGauge)
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c utils.Config) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	return &ReconcileReleaseManager{
		client: mgr.GetClient(),
		scheme: scheme,
		config: c,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	_, err := builder.SimpleController().
		WithManager(mgr).
		ForType(&picchuv1alpha1.ReleaseManager{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(_ event.UpdateEvent) bool { return false },
		}).
		Build(r)
	return err
}

var _ reconcile.Reconciler = &ReconcileReleaseManager{}

// ReconcileReleaseManager reconciles a ReleaseManager object
type ReconcileReleaseManager struct {
	client client.Client
	scheme *runtime.Scheme
	config utils.Config
}

// Reconcile reads that state of the cluster for a ReleaseManager object and makes changes based on the state read
// and what is in the ReleaseManager.Spec
func (r *ReconcileReleaseManager) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLog := clog.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLog.Info("Reconciling ReleaseManager")

	// Fetch the ReleaseManager instance
	rm := &picchuv1alpha1.ReleaseManager{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, rm); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	r.scheme.Default(rm)

	rmLog := reqLog.WithValues("App", rm.Spec.App, "Fleet", rm.Spec.Fleet, "Target", rm.Spec.Target)
	rmLog.Info("Reconciling Existing ReleaseManager")

	clusters, err := r.getClustersByFleet(rm.Namespace, rm.Spec.Fleet)
	if err != nil {
		rmLog.Error(err, "Failed to get clusters for fleet", "Fleet.Name", rm.Spec.Fleet)
	}
	var fleetSize uint32 = uint32(len(clusters))

	revisions, err := r.getRevisions(rmLog, request.Namespace, rm.Spec.Fleet, rm.Spec.App)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.scheme.Default(revisions)

	revisionStatuses := [][]picchuv1alpha1.ReleaseManagerRevisionStatus{}
	for i, cluster := range clusters {
		remoteClient, err := utils.RemoteClient(r.client, cluster)
		if err != nil {
			rmLog.Error(err, "Failed to create remote client", "Cluster.Name", cluster.Name)
			return reconcile.Result{}, err
		}

		ic := &IncarnationController{
			rrm:          r,
			remoteClient: remoteClient,
			logger:       rmLog.WithValues("Cluster", cluster.Name),
			rm:           rm,
			fs:           fleetSize,
		}

		incarnations := newIncarnationCollection(ic)

		for _, rev := range revisions.Items {
			incarnations.add(&rev)
		}

		syncer := ResourceSyncer{
			instance:     rm,
			incarnations: incarnations,
			cluster:      cluster,
			client:       remoteClient,
			reconciler:   r,
			log:          rmLog,
		}
		// -------------------------------------------------------------------------

		if !rm.IsDeleted() {
			rmLog.Info("Sync'ing releasemanager", "index", i, "count", fleetSize)
			rs, err := syncer.sync()
			if err != nil {
				return reconcile.Result{}, err
			}
			revisionStatuses = append(revisionStatuses, rs)
			continue
		}
		if !rm.IsFinalized() {
			rmLog.Info("Deleting releasemanager", "index", i, "count", fleetSize)
			err := syncer.del()
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	if !rm.IsDeleted() && len(revisionStatuses) > 0 {
		rm.Status.Revisions = r.resolveStatus(rmLog, revisionStatuses)
		if err := utils.UpdateStatus(context.TODO(), r.client, rm); err != nil {
			rmLog.Error(err, "Failed to update releasemanager status")
			return reconcile.Result{}, err
		}
		rmLog.Info("Updated releasemanager status", "Content", rm.Status, "Type", "ReleaseManager.Status")
		return reconcile.Result{RequeueAfter: r.config.RequeueAfter}, nil
	}
	if !rm.IsFinalized() {
		rm.Finalize()
		err := r.client.Update(context.TODO(), rm)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileReleaseManager) resolveStatus(log logr.Logger, statuses [][]picchuv1alpha1.ReleaseManagerRevisionStatus) []picchuv1alpha1.ReleaseManagerRevisionStatus {
	if len(statuses) == 1 {
		return statuses[0]
	}

	combined := statuses[0]
	for i, combinedRevStatus := range combined {
		for _, status := range statuses[1:] {
			for _, revStatus := range status {
				if revStatus.Tag == combinedRevStatus.Tag {
					combinedRevStatus.Scale.Current += revStatus.Scale.Current
					combinedRevStatus.Scale.Desired += revStatus.Scale.Desired
				}
			}
		}
		if combinedRevStatus.Scale.Current > combinedRevStatus.Scale.Peak {
			combinedRevStatus.Scale.Peak = combinedRevStatus.Scale.Current
		}
		combined[i] = combinedRevStatus
	}
	return combined
}

func (r *ReconcileReleaseManager) getRevisions(log logr.Logger, namespace, fleet, app string) (*picchuv1alpha1.RevisionList, error) {
	fleetLabel := fmt.Sprintf("%s%s", picchuv1alpha1.LabelFleetPrefix, fleet)
	log.Info("Looking for revisions")
	listOptions := client.
		InNamespace(namespace).
		MatchingLabels(map[string]string{
			picchuv1alpha1.LabelApp: app,
			fleetLabel:              "",
		})
	rl := &picchuv1alpha1.RevisionList{}
	err := r.client.List(context.TODO(), listOptions, rl)
	return rl, err
}

type ResourceSyncer struct {
	instance     *picchuv1alpha1.ReleaseManager
	incarnations *IncarnationCollection
	cluster      *picchuv1alpha1.Cluster
	client       client.Client
	reconciler   *ReconcileReleaseManager
	fleetSize    uint32
	log          logr.Logger
}

func (r *ResourceSyncer) getSecrets(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	secrets := &corev1.SecretList{}
	err := r.client.List(ctx, opts, secrets)
	if errors.IsNotFound(err) {
		return []runtime.Object{}, nil
	}
	return utils.MustExtractList(secrets), err
}

func (r *ResourceSyncer) getConfigMaps(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	configMaps := &corev1.ConfigMapList{}
	err := r.client.List(ctx, opts, configMaps)
	if errors.IsNotFound(err) {
		return []runtime.Object{}, nil
	}
	return utils.MustExtractList(configMaps), err
}

func (r *ResourceSyncer) sync() ([]picchuv1alpha1.ReleaseManagerRevisionStatus, error) {
	rs := []picchuv1alpha1.ReleaseManagerRevisionStatus{}

	// No more incarnations, delete myself
	if len(r.incarnations.revisioned()) == 0 {
		r.log.Info("No revisions found for releasemanager, deleting")
		return rs, r.reconciler.client.Delete(context.TODO(), r.instance)
	}

	if r.cluster.IsDeleted() {
		r.log.Info("Cluster is deleted, waiting for all incarnations to be deleted before finalizing")
		return rs, nil
	}

	if err := r.syncNamespace(); err != nil {
		return rs, err
	}
	if err := r.tickIncarnations(); err != nil {
		return rs, err
	}
	if err := r.syncApp(); err != nil {
		return rs, err
	}

	sorted := r.incarnations.sorted()
	for i := len(sorted) - 1; i >= 0; i-- {
		status := sorted[i].status
		if !(status.State.Current == "deleted" && sorted[i].revision == nil) {
			rs = append(rs, *status)
		}
	}
	return rs, nil
}

func (r *ResourceSyncer) del() error {
	return r.applyPlan("Delete App", &plan.DeleteApp{
		Namespace: r.instance.TargetNamespace(),
	})
}

func (r *ResourceSyncer) applyPlan(name string, p plan.Plan) error {
	r.log.Info("Applying plan", "Name", name, "Plan", p)
	return p.Apply(context.TODO(), r.client, r.log)
}

func (r *ResourceSyncer) syncNamespace() error {
	return r.applyPlan("Ensure Namespace", &plan.EnsureNamespace{
		Name:      r.instance.TargetNamespace(),
		OwnerName: r.instance.Name,
		OwnerType: picchuv1alpha1.OwnerReleaseManager,
	})
}

// SyncIncarnations syncs all revision deployments for this releasemanager
func (r *ResourceSyncer) tickIncarnations() error {
	r.log.Info("Incarnation count", "count", len(r.incarnations.sorted()))
	for _, incarnation := range r.incarnations.sorted() {
		sm := NewDeploymentStateManager(&incarnation)
		if err := sm.tick(); err != nil {
			return err
		}
		current := incarnation.status.State.Current
		if (current == "deployed" || current == "released") && incarnation.status.Metrics.GitDeploySeconds == nil {
			gitElapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
			incarnation.status.Metrics.GitDeploySeconds = &gitElapsed
			incarnationGitDeployLatency.Observe(gitElapsed)
			revElapsed := time.Since(incarnation.status.RevisionTimestamp.Time).Seconds()
			incarnation.status.Metrics.RevisionDeploySeconds = &revElapsed
			incarnationRevisionDeployLatency.Observe(revElapsed)
		}
		if current == "released" && incarnation.status.Metrics.GitReleaseSeconds == nil {
			elapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
			incarnation.status.Metrics.GitReleaseSeconds = &elapsed
			incarnationGitReleaseLatency.Observe(elapsed)
		}
		if current == "failed" && incarnation.status.Metrics.RevisionRollbackSeconds == nil {
			if incarnation.revision != nil {
				r.log.Info("observing rollback elapsed time")
				elapsed := incarnation.revision.SinceFailed().Seconds()
				incarnation.status.Metrics.RevisionRollbackSeconds = &elapsed
				incarnationRevisionRollbackLatency.Observe(elapsed)
			} else {
				r.log.Info("no revision rollback revision found")
			}
		}
	}
	return nil
}

func (r *ResourceSyncer) syncApp() error {
	portMap := map[string]picchuv1alpha1.PortInfo{}
	for _, incarnation := range r.incarnations.deployed() {
		for _, port := range incarnation.revision.Spec.Ports {
			_, ok := portMap[port.Name]
			if !ok {
				portMap[port.Name] = port
			}
		}
	}
	ports := make([]picchuv1alpha1.PortInfo, 0, len(portMap))
	for _, port := range portMap {
		ports = append(ports, port)
	}

	// Used to label Service and selector
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       r.instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}

	revisions, alertRules := r.prepareRevisionsAndRules()

	err := r.applyPlan("Sync Application", &plan.SyncApp{
		App:               r.instance.Spec.App,
		Namespace:         r.instance.TargetNamespace(),
		Labels:            labels,
		DefaultDomain:     r.cluster.Spec.DefaultDomain,
		PublicGateway:     r.cluster.Spec.Ingresses.Public.Gateway,
		PrivateGateway:    r.cluster.Spec.Ingresses.Private.Gateway,
		DeployedRevisions: revisions,
		AlertRules:        alertRules,
		Ports:             ports,
		TagRoutingHeader:  "",
		TrafficPolicy:     r.currentTrafficPolicy(),
	})
	if err != nil {
		return err
	}

	for _, revision := range revisions {
		revisionReleaseWeightGauge.
			With(prometheus.Labels{
				"app":            r.instance.Spec.App,
				"tag":            revision.Tag,
				"target":         r.instance.Spec.Target,
				"target_cluster": r.cluster.Name,
			}).
			Set(float64(revision.Weight))
	}
	return nil
}

// currentTrafficPolicy gets the latest releases traffic policy, or if there
// are no releases, then the latest revisions traffic policy.
func (r *ResourceSyncer) currentTrafficPolicy() *istiov1alpha3.TrafficPolicy {
	for _, incarnation := range r.incarnations.releasable() {
		if incarnation.revision != nil {
			return incarnation.revision.Spec.TrafficPolicy
		}
	}
	for _, incarnation := range r.incarnations.sorted() {
		if incarnation.revision != nil {
			return incarnation.revision.Spec.TrafficPolicy
		}
	}
	return nil
}

func (r *ResourceSyncer) prepareRevisionsAndRules() ([]plan.Revision, []monitoringv1.Rule) {
	alertRules := []monitoringv1.Rule{}

	if len(r.incarnations.deployed()) == 0 {
		return []plan.Revision{}, alertRules
	}

	revisionsMap := map[string]plan.Revision{}
	for _, i := range r.incarnations.deployed() {
		revisionsMap[i.tag] = plan.Revision{
			Tag:    i.tag,
			Weight: 0,
		}
	}

	// The idea here is we will work through releases from newest to oldest,
	// incrementing their weight if enough time has passed since their last
	// update, and stopping when we reach 100%. This will cause newer releases
	// to take from oldest fnord release.
	var percRemaining uint32 = 100
	// Tracking one route per port number
	incarnations := r.incarnations.releasable()
	count := len(incarnations)

	// setup alerts from latest release
	if count > 0 {
		alertRules = incarnations[0].target().AlertRules
	}

	for i, incarnation := range incarnations {
		status := incarnation.status
		oldCurrent := status.CurrentPercent
		current := incarnation.currentPercentTarget(percRemaining)
		if i+1 == count {
			current = percRemaining
		}
		incarnation.updateCurrentPercent(current)
		r.log.Info("CurrentPercentage Update", "Tag", incarnation.tag, "Old", oldCurrent, "Current", current)
		percRemaining -= current
		if percRemaining+current <= 0 {
			incarnation.setReleaseEligible(false)
		}
		revision := revisionsMap[incarnation.tag]
		revision.Weight = current
		revisionsMap[incarnation.tag] = revision
	}

	revisions := make([]plan.Revision, 0, len(revisionsMap))
	for _, revision := range revisionsMap {
		revisions = append(revisions, revision)
	}
	return revisions, alertRules
}

func (r *ReconcileReleaseManager) getClustersByFleet(namespace string, fleet string) ([]*picchuv1alpha1.Cluster, error) {
	clusterList := &picchuv1alpha1.ClusterList{}
	opts := client.
		MatchingLabels(map[string]string{picchuv1alpha1.LabelFleet: fleet}).
		InNamespace(namespace)
	err := r.client.List(context.TODO(), opts, clusterList)
	r.scheme.Default(clusterList)

	clusters := []*picchuv1alpha1.Cluster{}
	for _, cluster := range clusterList.Items {
		if !cluster.Spec.Enabled {
			continue
		}
		clusters = append(clusters, &cluster)
	}

	return clusters, err
}
