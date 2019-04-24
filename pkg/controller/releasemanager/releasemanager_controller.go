package releasemanager

import (
	"context"
	"fmt"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"
	promapi "go.medium.engineering/picchu/pkg/prometheus"

	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	StatusPort                 = "status"
	PrometheusScrapeLabel      = "prometheus.io/scrape"
	PrometheusScrapeLabelValue = "true"
)

var (
	log                       = logf.Log.WithName("controller_releasemanager")
	incarnationReleaseLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "picchu_incarnation_release_latency",
		Help: "track time from revision creation to incarnationrelease, " +
			"minus expected release time (release.rate.delay * math.Ceil(100.0 / release.rate.increment))",
		Buckets: prometheus.ExponentialBuckets(10, 3, 7),
	})
	incarnationDeployLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "picchu_incarnation_deploy_latency",
		Help:    "track time from revision creation to incarnation deploy",
		Buckets: prometheus.ExponentialBuckets(1, 3, 7),
	})
)

// Add creates a new ReleaseManager Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	metrics.Registry.MustRegister(incarnationDeployLatency)
	metrics.Registry.MustRegister(incarnationReleaseLatency)
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c utils.Config) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	api, err := promapi.NewAPI(c.PrometheusQueryAddress, c.PrometheusQueryTTL)
	if err != nil {
		panic(err)
	}
	return &ReconcileReleaseManager{
		client:  mgr.GetClient(),
		scheme:  scheme,
		config:  c,
		promAPI: api,
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
	promAPI *promapi.API
}

// Reconcile reads that state of the cluster for a ReleaseManager object and makes changes based on the state read
// and what is in the ReleaseManager.Spec
func (r *ReconcileReleaseManager) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
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

	cluster := &picchuv1alpha1.Cluster{}
	key := client.ObjectKey{request.Namespace, rm.Spec.Cluster}
	if err := r.client.Get(context.TODO(), key, cluster); err != nil {
		return reconcile.Result{}, err
	}
	r.scheme.Default(cluster)

	remoteClient, err := utils.RemoteClient(r.client, cluster)
	if err != nil {
		reqLog.Error(err, "Failed to create remote client", "Cluster.Key", key)
		return reconcile.Result{}, err
	}

	fleetSize, err := r.countFleetCohort(request.Namespace, cluster.Fleet())
	if err != nil {
		return reconcile.Result{}, err
	}

	ic := &IncarnationController{
		rrm:          r,
		remoteClient: remoteClient,
		logger:       reqLog,
		rm:           rm,
		fs:           fleetSize,
	}

	incarnations := newIncarnationCollection(ic)
	revisions, err := r.getRevisions(request.Namespace, cluster.Fleet(), rm.Spec.App)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.scheme.Default(revisions)
	for _, rev := range revisions.Items {
		incarnations.add(&rev)
	}
	incarnations.ensureValidRelease()

	aq := promapi.NewAlertQuery(rm.Spec.App)
	alertTags, err := r.promAPI.TaggedAlerts(context.TODO(), aq, time.Now())
	if err != nil {
		return reconcile.Result{}, err
	}
	alertTags, err := r.promAPI.TaggedAlerts(context.TODO(), promapi.NewAlertQuery(rm.Spec.App), time.Now())
	if err != nil {
		return reconcile.Result{}, err
	}
	syncer := ResourceSyncer{
		instance:     rm,
		incarnations: incarnations,
		cluster:      cluster,
		client:       remoteClient,
		reconciler:   r,
		log:          reqLog,
		alertTags:    alertTags,
	}
	// -------------------------------------------------------------------------

	if !rm.IsDeleted() {
		reqLog.Info("Sync'ing releasemanager")
		return syncer.sync()
	}
	if !rm.IsFinalized() {
		reqLog.Info("Deleting releasemanager")
		return syncer.del()
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileReleaseManager) getRevisions(namespace, fleet, app string) (*picchuv1alpha1.RevisionList, error) {
	fleetLabel := fmt.Sprintf("%s%s", picchuv1alpha1.LabelFleetPrefix, fleet)
	log.Info("Looking for revisions", "namespace", namespace, "fleet", fleet, "app", app)
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

func (r *ReconcileReleaseManager) countFleetCohort(namespace, fleet string) (uint32, error) {
	log.Info("Counting clusters in fleet cohort")
	listOptions := client.
		InNamespace(namespace).
		MatchingLabels(map[string]string{
			picchuv1alpha1.LabelFleet: fleet,
		})
	cl := &picchuv1alpha1.ClusterList{}
	err := r.client.List(context.TODO(), listOptions, cl)
	var cnt uint32 = 0
	for _, cluster := range cl.Items {
		for _, cond := range cluster.Status.Conditions {
			if cond.Name == "Ready" && cond.Status == "True" {
				cnt++
			}
		}
	}
	return cnt, err
}

type ResourceSyncer struct {
	instance     *picchuv1alpha1.ReleaseManager
	incarnations *IncarnationCollection
	cluster      *picchuv1alpha1.Cluster
	client       client.Client
	reconciler   *ReconcileReleaseManager
	fleetSize    uint32
	log          logr.Logger
	alertTags    []string
}

func (r *ResourceSyncer) getSecrets(ctx context.Context, opts *client.ListOptions) (*corev1.SecretList, error) {
	secrets := &corev1.SecretList{}
	err := r.client.List(ctx, opts, secrets)
	if errors.IsNotFound(err) {
		return secrets, nil
	}
	return secrets, err
}

func (r *ResourceSyncer) getConfigMaps(ctx context.Context, opts *client.ListOptions) (*corev1.ConfigMapList, error) {
	configMaps := &corev1.ConfigMapList{}
	err := r.client.List(ctx, opts, configMaps)
	if errors.IsNotFound(err) {
		return configMaps, nil
	}
	return configMaps, err
}

func (r *ResourceSyncer) sync() (reconcile.Result, error) {
	// No more incarnations, delete myself
	if len(r.incarnations.revisioned()) == 0 {
		r.log.Info("No revisions found for releasemanager, deleting")
		return reconcile.Result{}, r.reconciler.client.Delete(context.TODO(), r.instance)
	}

	if r.cluster.IsDeleted() {
		r.log.Info("Cluster is deleted, waiting for all incarnations to be deleted before finalizing")
		return reconcile.Result{}, nil
	}

	if err := r.syncNamespace(); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.tickIncarnations(); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.syncService(); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.syncDestinationRule(); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.syncVirtualService(); err != nil {
		return reconcile.Result{}, err
	}
	rs := []picchuv1alpha1.ReleaseManagerRevisionStatus{}
	sorted := r.incarnations.sorted()
	for i := len(sorted) - 1; i >= 0; i-- {
		status := sorted[i].status
		r.log.Info("Get status", "state", status.State)
		if !(status.State.Current == "deleted" && sorted[i].revision == nil) {
			rs = append(rs, *status)
		}
	}
	r.instance.Status.Revisions = rs
	if err := utils.UpdateStatus(context.TODO(), r.reconciler.client, r.instance); err != nil {
		r.log.Error(err, "Failed to update releasemanager status")
		return reconcile.Result{}, err
	}
	r.log.Info("Updated releasemanager status", "Content", r.instance.Status, "Type", "ReleaseManager.Status")
	return reconcile.Result{RequeueAfter: r.reconciler.config.RequeueAfter}, nil
}

func (r *ResourceSyncer) del() (reconcile.Result, error) {
	r.log.Info("Finalizing releasemanager")
	if err := r.deleteNamespace(); err != nil {
		return reconcile.Result{}, err
	}
	r.instance.Finalize()
	err := r.reconciler.client.Update(context.TODO(), r.instance)
	return reconcile.Result{}, err
}

func (r *ResourceSyncer) deleteNamespace() error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.instance.TargetNamespace(),
		},
	}
	// Might already be deleted
	if err := r.client.Delete(context.TODO(), namespace); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ResourceSyncer) syncNamespace() error {
	labels := map[string]string{
		"istio-injection":             "enabled",
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   r.instance.TargetNamespace(),
			Labels: labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, namespace, func(runtime.Object) error {
		return nil
	})
	r.log.Info("Namespace sync'd", "Op", op)
	return err
}

// SyncIncarnations syncs all revision deployments for this releasemanager
func (r *ResourceSyncer) tickIncarnations() error {
	r.log.Info("Incarnation count", "count", len(r.incarnations.sorted()))
	for _, incarnation := range r.incarnations.sorted() {
		oldState := incarnation.status.State

		r.log.Info("ok")
		sm := NewDeploymentStateManager(&incarnation)
		if err := sm.tick(); err != nil {
			return err
		}
		newState := incarnation.status.State
		if oldState.EqualTo(&newState) {
			continue
		}
		if newState.Current != newState.Target {
			continue
		}
		if newState.Target == "deployed" && incarnation.status.Metrics.DeploySeconds == nil {
			elapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
			incarnation.status.Metrics.DeploySeconds = &elapsed
			incarnationDeployLatency.Observe(elapsed)
		}
		if newState.Target == "released" && incarnation.status.Metrics.ReleaseSeconds == nil {
			elapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
			incarnation.status.Metrics.ReleaseSeconds = &elapsed
			incarnationReleaseLatency.Observe(elapsed)
		}
	}
	return nil
}

func (r *ResourceSyncer) syncService() error {
	portMap := map[string]corev1.ServicePort{}
	for _, incarnation := range r.incarnations.deployed() {
		if incarnation.revision == nil {
			continue
		}
		for _, port := range incarnation.revision.Spec.Ports {
			_, ok := portMap[port.Name]
			if !ok {
				portMap[port.Name] = corev1.ServicePort{
					Name:       port.Name,
					Protocol:   port.Protocol,
					Port:       port.Port,
					TargetPort: intstr.FromString(port.Name),
				}
			}
		}
	}
	if len(portMap) == 0 {
		return nil
	}
	ports := make([]corev1.ServicePort, 0, len(portMap))
	for _, port := range portMap {
		ports = append(ports, port)
	}

	// Used to label Service and selector
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       r.instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}
	if _, hasStatus := portMap[StatusPort]; hasStatus {
		labels[PrometheusScrapeLabel] = PrometheusScrapeLabelValue
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.instance.Spec.App,
			Namespace: r.instance.TargetNamespace(),
			Labels:    labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, service, func(runtime.Object) error {
		service.Spec.Ports = ports
		service.Spec.Selector = map[string]string{picchuv1alpha1.LabelApp: r.instance.Spec.App}
		return nil
	})

	r.log.Info("Service sync'd", "Op", op)
	return err
}

func (r *ResourceSyncer) syncDestinationRule() error {
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       r.instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}
	appName := r.instance.Spec.App
	service := fmt.Sprintf("%s.%s.svc.cluster.local", appName, r.instance.TargetNamespace())
	drule := &istiov1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.instance.Spec.App,
			Namespace: r.instance.TargetNamespace(),
			Labels:    labels,
		},
	}
	subsets := []istiov1alpha3.Subset{}
	for _, incarnation := range r.incarnations.deployed() {
		tag := incarnation.tag
		subsets = append(subsets, istiov1alpha3.Subset{
			Name:   tag,
			Labels: map[string]string{picchuv1alpha1.LabelTag: tag},
		})
	}
	spec := istiov1alpha3.DestinationRuleSpec{
		Host:    service,
		Subsets: subsets,
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, drule, func(runtime.Object) error {
		drule.Spec = spec
		return nil
	})
	r.log.Info("DestinationRule sync'd", "Type", "DestinationRule", "Audit", true, "Content", drule, "Op", op)
	return err
}

func (r *ResourceSyncer) syncVirtualService() error {
	if len(r.incarnations.deployed()) == 0 {
		return nil
	}
	appName := r.instance.Spec.App
	defaultDomain := r.cluster.Spec.DefaultDomain
	serviceHost := fmt.Sprintf("%s.%s.svc.cluster.local", appName, r.instance.TargetNamespace())
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       appName,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}
	defaultHost := fmt.Sprintf("%s.%s", r.instance.TargetNamespace(), defaultDomain)
	meshHost := fmt.Sprintf("%s.%s.svc.cluster.local", appName, r.instance.TargetNamespace())
	// keep a set of hosts
	hosts := map[string]bool{defaultHost: true, meshHost: true}
	publicGateway := r.cluster.Spec.Ingresses.Public.Gateway
	privateGateway := r.cluster.Spec.Ingresses.Private.Gateway
	gateways := []string{"mesh"}
	if publicGateway != "" {
		gateways = append(gateways, publicGateway)
	}
	if privateGateway != "" {
		gateways = append(gateways, privateGateway)
	}
	vs := &istiov1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: r.instance.TargetNamespace(),
			Labels:    labels,
		},
	}
	http := []istiov1alpha3.HTTPRoute{}

	// NOTE: We are iterating through the incarnations twice, these loops could
	// be combined if we sorted the incarnations by git timestamp at the expense
	// of readability

	// Incarnation specific releases are made for each port on private ingress
	if r.reconciler.config.TaggedRoutesEnabled {
		for _, incarnation := range r.incarnations.deployed() {
			http = append(http, incarnation.taggedRoutes(privateGateway, serviceHost)...)
		}
	}

	// The idea here is we will work through releases from newest to oldest,
	// incrementing their weight if enough time has passed since their last
	// update, and stopping when we reach 100%. This will cause newer releases
	// to take from oldest fnord release.
	var percRemaining uint32 = 100
	// Tracking one route per port number
	releaseRoutes := map[string]istiov1alpha3.HTTPRoute{}
	incarnations := r.incarnations.releasable()
	r.log.Info("Got my releases", "Count", len(incarnations))
	count := len(incarnations)
	// setup alerts from latest release
	if count > 0 {
		if err := incarnations[0].syncPrometheusRules(context.TODO()); err != nil {
			return err
		}
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

		tag := incarnation.tag

		if current > 0 && incarnation.revision != nil {
			for _, port := range incarnation.revision.Spec.Ports {
				portNumber := uint32(port.Port)
				filterHosts := port.Hosts
				gateway := []string{"mesh"}
				switch port.Mode {
				case picchuv1alpha1.PortPublic:
					if publicGateway == "" {
						r.log.Info("Can't configure publicGateway, undefined in Cluster")
						continue
					}
					gateway = []string{publicGateway}
					portNumber = uint32(port.IngressPort)
				case picchuv1alpha1.PortPrivate:
					if privateGateway == "" {
						r.log.Info("Can't configure privateGateway, undefined in Cluster")
						continue
					}
					gateway = []string{privateGateway}
					filterHosts = append(filterHosts, defaultHost)
					portNumber = uint32(port.IngressPort)
				}
				releaseRoute, ok := releaseRoutes[port.Name]
				if !ok {
					releaseRoute = istiov1alpha3.HTTPRoute{
						Redirect:              port.Istio.HTTP.Redirect,
						Rewrite:               port.Istio.HTTP.Rewrite,
						Retries:               port.Istio.HTTP.Retries,
						Fault:                 port.Istio.HTTP.Fault,
						Mirror:                port.Istio.HTTP.Mirror,
						AppendHeaders:         port.Istio.HTTP.AppendHeaders,
						RemoveResponseHeaders: port.Istio.HTTP.RemoveResponseHeaders,
					}
					for _, filterHost := range filterHosts {
						hosts[filterHost] = true
						releaseRoute.Match = append(releaseRoute.Match, istiov1alpha3.HTTPMatchRequest{
							Uri:       &istiocommonv1alpha1.StringMatch{Prefix: "/"},
							Authority: &istiocommonv1alpha1.StringMatch{Prefix: filterHost},
							Port:      portNumber,
							Gateways:  gateway,
						})
					}
					releaseRoute.Match = append(releaseRoute.Match, istiov1alpha3.HTTPMatchRequest{
						Uri:      &istiocommonv1alpha1.StringMatch{Prefix: "/"},
						Port:     uint32(port.Port),
						Gateways: []string{"mesh"},
					})
				}

				releaseRoute.Route = append(releaseRoute.Route, istiov1alpha3.DestinationWeight{
					Destination: istiov1alpha3.Destination{
						Host:   serviceHost,
						Port:   istiov1alpha3.PortSelector{Number: uint32(port.Port)},
						Subset: tag,
					},
					Weight: int(current),
				})
				releaseRoutes[port.Name] = releaseRoute
			}
		}
	}

	for _, route := range releaseRoutes {
		http = append(http, route)
	}

	// This can happen if there are released incarnations with deleted revisions
	// we want to wait until another revision is ready before updating.
	if len(http) < 1 {
		r.log.Info("Not sync'ing VirtualService, there are no valid releases")
		return nil
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, vs, func(runtime.Object) error {
		hostSlice := make([]string, 0, len(hosts))
		for h := range hosts {
			hostSlice = append(hostSlice, h)
		}
		vs.Spec.Hosts = hostSlice
		vs.Spec.Gateways = gateways
		vs.Spec.Http = http
		return nil
	})
	if err != nil {
		return err
	}

	r.log.Info("VirtualService sync'd", "Type", "VirtualService", "Audit", true, "Content", vs, "Op", op)
	return nil
}
