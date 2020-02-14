package releasemanager

import (
	"context"
	"fmt"
	"time"

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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	clog                         = logf.Log.WithName("controller_releasemanager")
	incarnationGitReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_release_latency",
		Help:    "track time from git revision creation to incarnation release",
		Buckets: prometheus.ExponentialBuckets(10, 3, 7),
	}, []string{"app", "target"})
	incarnationGitDeployLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_deploy_latency",
		Help:    "track time from git revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	}, []string{"app", "target"})
	incarnationGitCreateLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_git_create_latency",
		Help:    "track time from git revision creation to incarnation create",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	}, []string{"app", "target"})
	incarnationRevisionDeployLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_deploy_latency",
		Help:    "track time from revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	}, []string{"app", "target"})
	incarnationRevisionReleaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_release_latency",
		Help:    "track time from revision creation to incarnation release",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	}, []string{"app", "target"})
	revisionReleaseWeightGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_revision_release_weight",
		Help: "Percent of traffic a revision is getting as a target release",
	}, []string{"app", "target"})
	incarnationRevisionRollbackLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "picchu_revision_rollback_latency",
		Help:    "track time from failed revision to rollbacked incarnation",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	}, []string{"app", "target"})
	incarnationReleaseStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_incarnation_count",
		Help: "Number of incarnations in a state",
	}, []string{"app", "target", "state"})
	incarnationRevisionOldestStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_revision_oldest_age_seconds",
		Help: "The oldest revision in seconds for each state",
	}, []string{"app", "target", "state"})
)

// Add creates a new ReleaseManager Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	metrics.Registry.MustRegister(incarnationGitReleaseLatency)
	metrics.Registry.MustRegister(incarnationGitDeployLatency)
	metrics.Registry.MustRegister(incarnationGitCreateLatency)
	metrics.Registry.MustRegister(incarnationRevisionDeployLatency)
	metrics.Registry.MustRegister(incarnationRevisionRollbackLatency)
	metrics.Registry.MustRegister(incarnationRevisionReleaseLatency)
	metrics.Registry.MustRegister(incarnationReleaseStateGauge)
	metrics.Registry.MustRegister(incarnationRevisionOldestStateGauge)
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
	_, err := builder.ControllerManagedBy(mgr).
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
		return reconcile.Result{}, err
	}
	r.scheme.Default(rm)

	rmLog := reqLog.WithValues("App", rm.Spec.App, "Fleet", rm.Spec.Fleet, "Target", rm.Spec.Target)
	rmLog.Info("Reconciling Existing ReleaseManager")

	clusters, err := r.getClustersByFleet(ctx, rm.Namespace, rm.Spec.Fleet)
	if err != nil {
		rmLog.Error(err, "Failed to get clusters for fleet", "Fleet.Name", rm.Spec.Fleet)
	}
	clusterInfo := ClusterInfoList{}
	for _, cluster := range clusters {
		var scalingFactor float64 = 0.0
		if cluster.Spec.ScalingFactor != nil {
			scalingFactor = *cluster.Spec.ScalingFactor
		}
		clusterInfo = append(clusterInfo, ClusterInfo{
			Name:          cluster.Name,
			Live:          !cluster.Spec.HotStandby,
			ScalingFactor: scalingFactor,
		})
	}
	if clusterInfo.ClusterCount(true) == 0 {
		return reconcile.Result{}, nil
	}
	planApplier, err := r.newPlanApplier(ctx, rmLog, clusters)
	if err != nil {
		return reconcile.Result{}, err
	}

	deliveryClusters, err := r.getClustersByFleet(ctx, rm.Namespace, r.config.ServiceLevelsFleet)
	if err != nil {
		rmLog.Error(err, "Failed to get delivery clusters for fleet", "Fleet.Name", r.config.ServiceLevelsFleet)
	}
	deliveryClusterInfo := ClusterInfoList{}
	for _, cluster := range deliveryClusters {
		var scalingFactor float64 = 0.0
		if cluster.Spec.ScalingFactor != nil {
			scalingFactor = *cluster.Spec.ScalingFactor
		}
		deliveryClusterInfo = append(deliveryClusterInfo, ClusterInfo{
			Name:          cluster.Name,
			Live:          !cluster.Spec.HotStandby,
			ScalingFactor: scalingFactor,
		})
	}

	deliveryApplier, err := r.newPlanApplier(ctx, rmLog, deliveryClusters)
	if err != nil {
		return reconcile.Result{}, err
	}

	observer, err := r.newObserver(ctx, rmLog, clusters)
	if err != nil {
		return reconcile.Result{}, err
	}

	revisions, err := r.getRevisions(ctx, rmLog, request.Namespace, rm.Spec.Fleet, rm.Spec.App, rm.Spec.Target)
	if err != nil {
		return reconcile.Result{}, err
	}

	observation, err := observer.Observe(ctx, rm.TargetNamespace())
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterConfig, err := r.getClusterConfig(clusters)
	if err != nil {
		return reconcile.Result{}, err
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
		clusterConfig:   clusterConfig,
		picchuConfig:    r.config,
	}

	if !rm.IsDeleted() {
		rmLog.Info("Sync'ing releasemanager")
		rs, err := syncer.sync(ctx)
		if err != nil {
			return reconcile.Result{}, err
		}
		rm.Status.Revisions = rs
		if err := utils.UpdateStatus(ctx, r.client, rm); err != nil {
			rmLog.Error(err, "Failed to update releasemanager status")
			return reconcile.Result{}, err
		}
		rmLog.Info("Updated releasemanager status", "Content", rm.Status, "Type", "ReleaseManager.Status")
		return reconcile.Result{RequeueAfter: r.config.RequeueAfter}, nil
	} else if !rm.IsFinalized() {
		rmLog.Info("Deleting ServiceLevels")
		if err := syncer.delServiceLevels(ctx); err != nil {
			return reconcile.Result{}, err
		}

		rmLog.Info("Deleting releasemanager")
		if err := syncer.del(ctx); err != nil {
			return reconcile.Result{}, err
		}

		rm.Finalize()
		err := r.client.Update(ctx, rm)
		return reconcile.Result{}, err
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
		rev := rl.Items[i]
		if rev.HasTarget(target) {
			withTargets.Items = append(withTargets.Items, rev)
		}
	}
	return withTargets, nil
}

func (r *ReconcileReleaseManager) getClustersByFleet(ctx context.Context, namespace string, fleet string) ([]picchuv1alpha1.Cluster, error) {
	clusterList := &picchuv1alpha1.ClusterList{}
	opts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{picchuv1alpha1.LabelFleet: fleet}),
	}
	err := r.client.List(ctx, clusterList, opts)
	r.scheme.Default(clusterList)

	clusters := []picchuv1alpha1.Cluster{}
	for i := range clusterList.Items {
		cluster := clusterList.Items[i]
		if !cluster.Spec.Enabled {
			continue
		}
		clusters = append(clusters, cluster)
	}

	return clusters, err
}

// ClusterConfig is the cluster information needed to create cluster specific resources
type ClusterConfig struct {
	DefaultDomain         string
	PublicIngressGateway  string
	PrivateIngressGateway string
}

// Ensure all clusters share the same config and return
func (r *ReconcileReleaseManager) getClusterConfig(clusters []picchuv1alpha1.Cluster) (ClusterConfig, error) {
	spec := clusters[0].Spec
	c := ClusterConfig{
		DefaultDomain:         spec.DefaultDomain,
		PublicIngressGateway:  spec.Ingresses.Public.Gateway,
		PrivateIngressGateway: spec.Ingresses.Private.Gateway,
	}
	for i := range clusters[1:] {
		spec = clusters[i].Spec
		if c.DefaultDomain != spec.DefaultDomain {
			return c, fmt.Errorf("Default domains in fleet don't match")
		}
		if c.PublicIngressGateway != spec.Ingresses.Public.Gateway {
			return c, fmt.Errorf("Public ingress gateways in fleet don't match")
		}
		if c.PrivateIngressGateway != spec.Ingresses.Private.Gateway {
			return c, fmt.Errorf("Private ingress gateways in fleet don't match")
		}
	}
	return c, nil
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
			scalingFactor := cluster.Spec.ScalingFactor
			if scalingFactor == nil || *scalingFactor < 0.1 {
				panic("Refusing to scale lower than 0.1 on a cluster")
			}
			appliers[i] = plan.NewClusterApplier(remoteClient, *scalingFactor, log.WithValues("Cluster", cluster.Name))
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
