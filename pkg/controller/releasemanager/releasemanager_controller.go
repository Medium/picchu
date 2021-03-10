package releasemanager

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/observe"
	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	clog                        = logf.Log.WithName("controller_releasemanager")
	incarnationGitCreateLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_create_latency",
		Help:    "track time from git revision creation to incarnation create",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationGitDeployLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_deploy_latency",
		Help:    "track time from git revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationGitCanaryLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_canary_latency",
		Help:    "track time from git revision creation to incarnation canary",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationGitPendingReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_pending_release_latency",
		Help:    "track time from git revision creation to incarnation pendingRelease",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationGitReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_release_latency",
		Help:    "track time from git revision creation to incarnation release",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationRevisionDeployLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_deploy_latency",
		Help:    "track time from revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationRevisionCanaryLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_canary_latency",
		Help:    "track time from revision creation to incarnation canary",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationRevisionPendingReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_pending_release_latency",
		Help:    "track time from revision creation to incarnation pendingRelease",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	incarnationRevisionReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_release_latency",
		Help:    "track time from revision creation to incarnation release",
		Buckets: prometheus.ExponentialBuckets(120, 3, 7),
	}, []string{"app", "target"})
	revisionReleaseWeightGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_revision_release_weight",
		Help: "Percent of traffic a revision is getting as a target release",
	}, []string{"app", "target"})
	incarnationRevisionRollbackLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_rollback_latency",
		Help:    "track time from failed revision to rollbacked incarnation",
		Buckets: prometheus.ExponentialBuckets(30, 3, 7),
	}, []string{"app", "target"})
	incarnationReleaseStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_incarnation_count",
		Help: "Number of incarnations in a state",
	}, []string{"app", "target", "state"})
	incarnationRevisionOldestStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_revision_oldest_age_seconds",
		Help: "The oldest revision in seconds for each state",
	}, []string{"app", "target", "state"})
	incarnationDeployLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_deploy_latency",
		Help:    "track time a revision spends from deploying to deployed",
		Buckets: prometheus.ExponentialBuckets(30, 3, 7),
	}, []string{"app", "target"})
	incarnationCanaryLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_canary_latency",
		Help:    "track time a revision spends from canarying to canaried",
		Buckets: prometheus.ExponentialBuckets(30, 3, 7),
	}, []string{"app", "target"})
	incarnationReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_release_latency",
		Help:    "track time a revision spends from releasing to released",
		Buckets: prometheus.ExponentialBuckets(30, 3, 7),
	}, []string{"app", "target"})
	reconcileInterval = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_release_manager_reconcile_interval",
		Help:    "track time between last update and reconcile request",
		Buckets: []float64{10, 15, 20, 30, 45, 60, 90, 120},
	}, []string{"app", "target"})
)

// Add creates a new ReleaseManager Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	metrics.Registry.MustRegister(
		incarnationGitCreateLatency,
		incarnationGitDeployLatency,
		incarnationGitCanaryLatency,
		incarnationGitPendingReleaseLatency,
		incarnationGitReleaseLatency,
		incarnationRevisionDeployLatency,
		incarnationRevisionCanaryLatency,
		incarnationRevisionPendingReleaseLatency,
		incarnationRevisionReleaseLatency,
		incarnationRevisionRollbackLatency,
		incarnationDeployLatency,
		incarnationCanaryLatency,
		incarnationReleaseLatency,
		incarnationReleaseStateGauge,
		incarnationRevisionOldestStateGauge,
		revisionReleaseWeightGauge,
		reconcileInterval,
	)
	return add(mgr, newReconciler(mgr, c), c)
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
func add(mgr manager.Manager, r reconcile.Reconciler, c utils.Config) error {
	_, err := builder.ControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: c.ConcurrentReleaseManagers}).
		For(&picchuv1alpha1.ReleaseManager{}).
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
	start := time.Now()
	traceID := uuid.New().String()
	reqLog := clog.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "Trace", traceID)
	defer func() {
		reqLog.Info("Finished releasemanager reconcile", "Elapsed", time.Since(start))
	}()
	reqLog.Info("Reconciling ReleaseManager")
	ctx := context.TODO()

	// Fetch the ReleaseManager instance
	rm := &picchuv1alpha1.ReleaseManager{}
	if err := r.client.Get(ctx, request.NamespacedName, rm); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return r.requeue(reqLog, err)
	}
	r.scheme.Default(rm)

	rmLog := reqLog.WithValues("App", rm.Spec.App, "Fleet", rm.Spec.Fleet, "Target", rm.Spec.Target)
	for label := range rm.Labels {
		if label == picchuv1alpha1.LabelIgnore {
			rmLog.Info("Ignoring ReleaseManager")
			return r.requeue(rmLog, nil)
		}
	}
	rmLog.Info("Reconciling Existing ReleaseManager")

	if lastUpdated := rm.Status.LastUpdated; lastUpdated != nil {
		reconcileInterval.With(prometheus.Labels{
			"app":    rm.Spec.App,
			"target": rm.Spec.Target,
		}).Observe(start.Sub(lastUpdated.Time).Seconds())
	}

	clusters, err := r.getClustersByFleet(ctx, rm.Namespace, rm.Spec.Fleet)
	if err != nil {
		return r.requeue(rmLog, fmt.Errorf("failed to get clusters for fleet %s: %w", rm.Spec.Fleet, err))
	}
	clusterInfo := ClusterInfoList{}
	for _, cluster := range clusters {
		var scalingFactor = 0.0
		if cluster.Spec.ScalingFactor != nil {
			scalingFactor = *cluster.Spec.ScalingFactor
		}
		if cluster.Spec.ScalingFactorString != nil {
			f, err := strconv.ParseFloat(*cluster.Spec.ScalingFactorString, 64)
			if err != nil {
				reqLog.Error(err, "Could not parse %v to float", *cluster.Spec.ScalingFactorString)
			} else {
				scalingFactor = f
			}
		}
		clusterInfo = append(clusterInfo, ClusterInfo{
			Name:          cluster.Name,
			Live:          !cluster.Spec.HotStandby,
			ScalingFactor: scalingFactor,
		})
	}
	if clusterInfo.ClusterCount(true) == 0 {
		return r.requeue(rmLog, err)
	}
	planApplier, err := r.newPlanApplier(ctx, rmLog, clusters)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	deliveryClusters, err := r.getClustersByFleet(ctx, rm.Namespace, r.config.ServiceLevelsFleet)
	if err != nil {
		return r.requeue(rmLog, fmt.Errorf("failed to get delivery clusters for fleet %s: %w", r.config.ServiceLevelsFleet, err))
	}
	deliveryClusterInfo := ClusterInfoList{}
	for _, cluster := range deliveryClusters {
		var scalingFactor = 0.0
		if cluster.Spec.ScalingFactor != nil {
			scalingFactor = *cluster.Spec.ScalingFactor
		}
		if cluster.Spec.ScalingFactorString != nil {
			f, err := strconv.ParseFloat(*cluster.Spec.ScalingFactorString, 64)
			if err != nil {
				reqLog.Error(err, "Could not parse %v to float", *cluster.Spec.ScalingFactorString)
			} else {
				scalingFactor = f
			}
		}
		deliveryClusterInfo = append(deliveryClusterInfo, ClusterInfo{
			Name:          cluster.Name,
			Live:          !cluster.Spec.HotStandby,
			ScalingFactor: scalingFactor,
		})
	}

	deliveryApplier, err := r.newPlanApplier(ctx, rmLog, deliveryClusters)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	observer, err := r.newObserver(ctx, rmLog, clusters)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	revisions, err := r.getRevisions(ctx, rmLog, request.Namespace, rm.Spec.Fleet, rm.Spec.App, rm.Spec.Target)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	faults, err := r.getFaults(ctx, rmLog, request.Namespace, rm.Spec.App, rm.Spec.Target)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	observation, err := observer.Observe(ctx, rm.TargetNamespace())
	if err != nil {
		return r.requeue(rmLog, err)
	}

	ic := &IncarnationController{
		deliveryClient:  r.client,
		deliveryApplier: deliveryApplier,
		planApplier:     planApplier,
		log:             rmLog,
		releaseManager:  rm,
		clusterInfo:     clusterInfo,
	}

	syncer := ResourceSyncer{
		deliveryClient:  r.client,
		deliveryApplier: deliveryApplier,
		planApplier:     planApplier,
		observer:        observer,
		instance:        rm,
		incarnations:    newIncarnationCollection(ic, revisions, observation, r.config),
		reconciler:      r,
		log:             rmLog,
		picchuConfig:    r.config,
		faults:          faults,
	}

	if !rm.IsDeleted() {
		rmLog.Info("Sync'ing releasemanager")
		rs, err := syncer.sync(ctx)
		if err != nil {
			return r.requeue(rmLog, err)
		}

		// TODO(bob): Figure out why we are getting object modified errors
		// Refetch fresh ReleaseManager instance so we don't get conflicting updates
		srm := &picchuv1alpha1.ReleaseManager{}
		if err := r.client.Get(ctx, request.NamespacedName, srm); err != nil {
			if errors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				return reconcile.Result{}, nil
			}
			// Error reading the object - requeue the request.
			return r.requeue(reqLog, err)
		}
		r.scheme.Default(srm)
		srm.Status.Revisions = rs
		timeNow := metav1.NewTime(time.Now())
		srm.Status.LastUpdated = &timeNow
		if err := utils.UpdateStatus(ctx, r.client, srm); err != nil {
			return r.requeue(rmLog, err)
		}
		return r.requeue(rmLog, nil)
	} else if !rm.IsFinalized() {
		rmLog.Info("Deleting ServiceLevels")
		if err := syncer.delServiceLevels(ctx); err != nil {
			return r.requeue(rmLog, err)
		}

		rmLog.Info("Deleting releasemanager")
		if err := syncer.del(ctx); err != nil {
			return r.requeue(rmLog, err)
		}

		rm.Finalize()
		err := r.client.Update(ctx, rm)
		return r.requeue(rmLog, err)
	}

	rmLog.Info("ReleaseManager is deleted and finalized")
	return reconcile.Result{}, nil
}

func (r *ReconcileReleaseManager) getRevisions(ctx context.Context, log logr.Logger, namespace, fleet, app, target string) (*picchuv1alpha1.RevisionList, error) {
	fleetLabel := fmt.Sprintf("%s%s", picchuv1alpha1.LabelFleetPrefix, fleet)
	log.Info("Looking for revisions")
	listOptions := []client.ListOption{
		&client.ListOptions{Namespace: namespace},
		client.MatchingLabels{
			picchuv1alpha1.LabelApp: app,
			fleetLabel:              ""},
	}
	rl := &picchuv1alpha1.RevisionList{}
	err := r.client.List(ctx, rl, listOptions...)
	if err != nil {
		return nil, err
	}
	r.scheme.Default(rl)
	withTargets := &picchuv1alpha1.RevisionList{}
	for i := range rl.Items {
		for j := range rl.Items[i].Spec.Targets {
			if rl.Items[i].Spec.Targets[j].Release.TTL == 0 {
				panic("Defaults weren't set")
			}
		}
		rev := rl.Items[i]
		if rev.HasTarget(target) {
			withTargets.Items = append(withTargets.Items, rev)
		}
	}
	return withTargets, nil
}

func (r *ReconcileReleaseManager) getFaults(ctx context.Context, log logr.Logger, namespace, app, target string) ([]picchuv1alpha1.HTTPPortFault, error) {
	log.Info("Looking for faultInjectors")

	selector, err := labels.Parse(fmt.Sprintf("%s=%s,%s in (,%s)", picchuv1alpha1.LabelApp, app, picchuv1alpha1.LabelTarget, target))
	if err != nil {
		log.Error(err, "Failed to parse label requirement")
		return nil, err
	}
	listOptions := []client.ListOption{
		client.InNamespace(namespace),
		&client.MatchingLabelsSelector{Selector: selector},
	}

	list := &picchuv1alpha1.FaultInjectorList{}
	err = r.client.List(ctx, list, listOptions...)
	if err != nil {
		log.Error(err, "Failed to list faultInjectors")
		return nil, err
	}
	r.scheme.Default(list)

	var faults []picchuv1alpha1.HTTPPortFault
	for _, item := range list.Items {
		faults = append(faults, item.Spec.HTTPPortFaults...)
	}

	return faults, nil
}

// requeue gives us a chance to log the error with our trace-id and stacktrace to make debugging simpler
func (r *ReconcileReleaseManager) requeue(log logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		log.Error(err, "An error occurred during reconciliation")
		return reconcile.Result{}, err
	}
	log.Info("Requeuing", "duration", r.config.RequeueAfter)
	return reconcile.Result{RequeueAfter: r.config.RequeueAfter}, nil
}

func (r *ReconcileReleaseManager) getClustersByFleet(ctx context.Context, namespace string, fleet string) ([]picchuv1alpha1.Cluster, error) {
	clusterList := &picchuv1alpha1.ClusterList{}
	opts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{picchuv1alpha1.LabelFleet: fleet}),
	}
	err := r.client.List(ctx, clusterList, opts)
	r.scheme.Default(clusterList)

	var clusters []picchuv1alpha1.Cluster
	for i := range clusterList.Items {
		cluster := clusterList.Items[i]
		if !cluster.Spec.Enabled {
			continue
		}
		clusters = append(clusters, cluster)
	}

	return clusters, err
}

func (r *ReconcileReleaseManager) newPlanApplier(ctx context.Context, log logr.Logger, clusters []picchuv1alpha1.Cluster) (plan.Applier, error) {
	g, ctx := errgroup.WithContext(ctx)
	appliers := make([]plan.Applier, len(clusters))
	for i := range clusters {
		i := i
		cluster := clusters[i]
		g.Go(func() error {
			remoteClient, err := utils.RemoteClient(ctx, log, r.client, &cluster)
			if err != nil {
				log.Error(err, "Failed to create remote client")
				return err
			}

			appliers[i] = plan.NewClusterApplier(remoteClient, &cluster, log.WithValues("Cluster", cluster.Name))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return plan.NewConcurrentApplier(appliers, log), nil
}

func (r *ReconcileReleaseManager) newObserver(ctx context.Context, log logr.Logger, clusters []picchuv1alpha1.Cluster) (observe.Observer, error) {
	g, ctx := errgroup.WithContext(ctx)
	observers := make([]observe.Observer, len(clusters))
	for i := range clusters {
		i := i
		cluster := clusters[i]
		g.Go(func() error {
			remoteClient, err := utils.RemoteClient(ctx, log, r.client, &cluster)
			if err != nil {
				log.Error(err, "Failed to create remote client")
				return err
			}
			observerCluster := observe.Cluster{Name: cluster.Name, Live: !cluster.Spec.HotStandby}
			observers[i] = observe.NewClusterObserver(observerCluster, remoteClient, log.WithValues("Cluster", cluster.Name))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return observe.NewConcurrentObserver(observers, log), nil
}
