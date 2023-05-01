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
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/observe"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	rmClog                      = logf.Log.WithName("controller_releasemanager")
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
	}, []string{"app", "target", "state", "revision"})
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

// ReleaseManagerReconciler reconciles a ReleaseManager object
type ReleaseManagerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config utils.Config
}

// +kubebuilder:rbac:groups=picchu.medium.engineering,resources=releasemanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=picchu.medium.engineering,resources=releasemanagers/status,verbs=get;update;patch

func (r *ReleaseManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	start := time.Now()
	traceID := uuid.New().String()
	_ = context.Background()
	reqLog := r.Log.WithValues("releasemanager", req.NamespacedName, "Trace", traceID)

	// your logic here

	rm := &picchuv1alpha1.ReleaseManager{}
	if err := r.Client.Get(ctx, req.NamespacedName, rm); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return r.requeue(reqLog, err)
	}

	r.Scheme.Default(rm)

	rmLog := reqLog.WithValues("App", rm.Spec.App, "Fleet", rm.Spec.Fleet, "Target", rm.Spec.Target)

	for label := range rm.Labels {
		if label == picchuv1alpha1.LabelIgnore {
			rmLog.Info("Ignoring ReleaseManager")
			return r.requeue(rmLog, nil)
		}
	}

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

	deliveryClusters, err := r.getClustersByFleet(ctx, rm.Namespace, r.Config.ServiceLevelsFleet)
	if err != nil {
		return r.requeue(rmLog, fmt.Errorf("failed to get delivery clusters for fleet %s: %w", r.Config.ServiceLevelsFleet, err))
	}
	deliveryClusterInfo := ClusterInfoList{}
	for _, cluster := range deliveryClusters {
		var scalingFactor = 0.0
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

	revisions, err := r.getRevisions(ctx, rmLog, req.Namespace, rm.Spec.Fleet, rm.Spec.App, rm.Spec.Target)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	faults, err := r.getFaults(ctx, rmLog, req.Namespace, rm.Spec.App, rm.Spec.Target)
	if err != nil {
		return r.requeue(rmLog, err)
	}

	observation, err := observer.Observe(ctx, rm.TargetNamespace())
	if err != nil {
		return r.requeue(rmLog, err)
	}

	ic := &IncarnationController{
		deliveryClient:  r.Client,
		deliveryApplier: deliveryApplier,
		planApplier:     planApplier,
		log:             rmLog,
		releaseManager:  rm,
		clusterInfo:     clusterInfo,
	}

	syncer := ResourceSyncer{
		deliveryClient:  r.Client,
		deliveryApplier: deliveryApplier,
		planApplier:     planApplier,
		observer:        observer,
		instance:        rm,
		incarnations:    newIncarnationCollection(ic, revisions, observation, r.Config),
		reconciler:      r,
		log:             rmLog,
		picchuConfig:    r.Config,
		faults:          faults,
	}

	if !rm.IsDeleted() {
		rs, err := syncer.sync(ctx)
		if err != nil {
			return r.requeue(rmLog, err)
		}

		// TODO(bob): Figure out why we are getting object modified errors
		// Refetch fresh ReleaseManager instance so we don't get conflicting updates
		srm := &picchuv1alpha1.ReleaseManager{}
		if err := r.Client.Get(ctx, req.NamespacedName, srm); err != nil {
			if errors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				return reconcile.Result{}, nil
			}
			// Error reading the object - requeue the request.
			return r.requeue(reqLog, err)
		}
		r.Scheme.Default(srm)
		srm.Status.Revisions = rs
		timeNow := metav1.NewTime(time.Now())
		srm.Status.LastUpdated = &timeNow
		if err := utils.UpdateStatus(ctx, r.Client, srm); err != nil {
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
		err := r.Client.Update(ctx, rm)
		return r.requeue(rmLog, err)
	}

	rmLog.Info("ReleaseManager is deleted and finalized")
	return ctrl.Result{}, nil
}

func (r *ReleaseManagerReconciler) getRevisions(ctx context.Context, log logr.Logger, namespace, fleet, app, target string) (*picchuv1alpha1.RevisionList, error) {
	fleetLabel := fmt.Sprintf("%s%s", picchuv1alpha1.LabelFleetPrefix, fleet)
	listOptions := []client.ListOption{
		&client.ListOptions{Namespace: namespace},
		client.MatchingLabels{
			picchuv1alpha1.LabelApp: app,
			fleetLabel:              ""},
	}
	rl := &picchuv1alpha1.RevisionList{}
	err := r.Client.List(ctx, rl, listOptions...)
	if err != nil {
		return nil, err
	}
	r.Scheme.Default(rl)
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

func (r *ReleaseManagerReconciler) getFaults(ctx context.Context, log logr.Logger, namespace, app, target string) ([]picchuv1alpha1.HTTPPortFault, error) {
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
	err = r.Client.List(ctx, list, listOptions...)
	if err != nil {
		log.Error(err, "Failed to list faultInjectors")
		return nil, err
	}
	r.Scheme.Default(list)

	var faults []picchuv1alpha1.HTTPPortFault
	for _, item := range list.Items {
		faults = append(faults, item.Spec.HTTPPortFaults...)
	}

	return faults, nil
}

// requeue gives us a chance to log the error with our trace-id and stacktrace to make debugging simpler
func (r *ReleaseManagerReconciler) requeue(log logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		log.Error(err, "An error occurred during reconciliation")
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: r.Config.RequeueAfter}, nil
}

func (r *ReleaseManagerReconciler) getClustersByFleet(ctx context.Context, namespace string, fleet string) ([]picchuv1alpha1.Cluster, error) {
	clusterList := &picchuv1alpha1.ClusterList{}
	opts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{picchuv1alpha1.LabelFleet: fleet}),
	}
	err := r.Client.List(ctx, clusterList, opts)
	r.Scheme.Default(clusterList)

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

func (r *ReleaseManagerReconciler) newPlanApplier(ctx context.Context, log logr.Logger, clusters []picchuv1alpha1.Cluster) (plan.Applier, error) {
	g, ctx := errgroup.WithContext(ctx)
	appliers := make([]plan.Applier, len(clusters))
	for i := range clusters {
		i := i
		cluster := clusters[i]
		g.Go(func() error {
			remoteClient, err := utils.RemoteClient(ctx, log, r.Client, &cluster)
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

func (r *ReleaseManagerReconciler) newObserver(ctx context.Context, log logr.Logger, clusters []picchuv1alpha1.Cluster) (observe.Observer, error) {
	g, ctx := errgroup.WithContext(ctx)
	observers := make([]observe.Observer, len(clusters))
	for i := range clusters {
		i := i
		cluster := clusters[i]
		g.Go(func() error {
			remoteClient, err := utils.RemoteClient(ctx, log, r.Client, &cluster)
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

func (r *ReleaseManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.Config.ConcurrentReleaseManagers}).
		For(&picchuv1alpha1.ReleaseManager{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldObject := e.ObjectOld.(*picchuv1alpha1.ReleaseManager)
				newObject := e.ObjectNew.(*picchuv1alpha1.ReleaseManager)
				return oldObject.Status.LastUpdated.Equal(newObject.Status.LastUpdated)
			},
		}).
		Complete(r)
}
