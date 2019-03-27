package releasemanager

import (
	"context"
	"fmt"
	"math"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log                       = logf.Log.WithName("controller_releasemanager")
	incarnationReleaseLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "picchu_incarnation_release_latency",
		Help: "track time from revision creation to incarnationrelease, " +
			"minus expected release time (release.rate.delay * math.Ceil(100.0 / release.rate.increment))",
		Buckets: prometheus.ExponentialBuckets(10, 3, 7),
	})
)

// Add creates a new ReleaseManager Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	metrics.Registry.MustRegister(incarnationReleaseLatency)
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c utils.Config) reconcile.Reconciler {
	return &ReconcileReleaseManager{client: mgr.GetClient(), scheme: mgr.GetScheme(), config: c}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("releasemanager-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ReleaseManager
	err = c.Watch(&source.Kind{Type: &picchuv1alpha1.ReleaseManager{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Incarnations and Clusters and requeue the owner ReleaseManager
	err = c.Watch(&source.Kind{Type: &picchuv1alpha1.Incarnation{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &picchuv1alpha1.ReleaseManager{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &picchuv1alpha1.Cluster{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &picchuv1alpha1.ReleaseManager{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileReleaseManager{}

// ReconcileReleaseManager reconciles a ReleaseManager object
type ReconcileReleaseManager struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config utils.Config
}

// Reconcile reads that state of the cluster for a ReleaseManager object and makes changes based on the state read
// and what is in the ReleaseManager.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileReleaseManager) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ReleaseManager")

	// Fetch the ReleaseManager instance
	instance := &picchuv1alpha1.ReleaseManager{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
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

	cluster := &picchuv1alpha1.Cluster{}
	key := client.ObjectKey{request.Namespace, instance.Spec.Cluster}
	if err := r.client.Get(context.TODO(), key, cluster); err != nil {
		return reconcile.Result{}, err
	}
	r.scheme.Default(cluster)
	incarnationList := &picchuv1alpha1.IncarnationList{}
	labelSelector := client.MatchingLabels(map[string]string{
		picchuv1alpha1.LabelApp:     instance.Spec.App,
		picchuv1alpha1.LabelCluster: instance.Spec.Cluster,
	})
	if err := r.client.List(context.TODO(), labelSelector, incarnationList); err != nil {
		return reconcile.Result{}, err
	}
	r.scheme.Default(incarnationList)

	remoteClient, err := utils.RemoteClient(r.client, cluster)
	if err != nil {
		reqLogger.Error(err, "Failed to create remote client", "Cluster.Key", key)
		return reconcile.Result{}, err
	}
	syncer := ResourceSyncer{
		Instance:        instance,
		IncarnationList: incarnationList,
		Cluster:         cluster,
		Client:          remoteClient,
		PicchuClient:    r.client,
	}
	if !instance.IsDeleted() {
		// No more incarnations, delete myself
		if len(incarnationList.Items) == 0 {
			reqLogger.Info("No incarnations found for releasemanager, deleting")
			return reconcile.Result{}, r.client.Delete(context.TODO(), instance)
		}

		if cluster.IsDeleted() {
			reqLogger.Info("Cluster is deleted, waiting for all incarnations to be deleted before finalizing")
			return reconcile.Result{RequeueAfter: r.config.RequeueAfter}, nil
		}

		if err := syncer.SyncNamespace(); err != nil {
			return reconcile.Result{}, err
		}
		if err := syncer.SyncService(); err != nil {
			return reconcile.Result{}, err
		}
		if err := syncer.SyncDestinationRule(); err != nil {
			return reconcile.Result{}, err
		}
		if err := syncer.SyncVirtualService(); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{RequeueAfter: r.config.RequeueAfter}, nil
	}
	if !instance.IsFinalized() {
		reqLogger.Info("Finalizing releasemanager")
		if err := syncer.DeleteNamespace(); err != nil {
			return reconcile.Result{}, err
		}
		instance.Finalize()
		if err := r.client.Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

type ResourceSyncer struct {
	Instance        *picchuv1alpha1.ReleaseManager
	IncarnationList *picchuv1alpha1.IncarnationList
	Cluster         *picchuv1alpha1.Cluster
	Client          client.Client
	PicchuClient    client.Client
}

func (r *ResourceSyncer) DeleteNamespace() error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Instance.TargetNamespace(),
		},
	}
	// Might already be deleted
	if err := r.Client.Delete(context.TODO(), namespace); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ResourceSyncer) SyncNamespace() error {
	labels := map[string]string{
		"istio-injection":             "enabled",
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.Instance.Name,
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   r.Instance.TargetNamespace(),
			Labels: labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, namespace, func(runtime.Object) error {
		namespace.ObjectMeta.SetLabels(labels)
		return nil
	})
	log.Info("Namespace sync'd", "Op", op)
	return err
}

func (r *ResourceSyncer) SyncService() error {
	portMap := map[string]corev1.ServicePort{}
	for _, incarnation := range r.IncarnationList.Items {
		for _, port := range incarnation.Spec.Ports {
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
	ports := make([]corev1.ServicePort, 0, len(portMap))
	for _, port := range portMap {
		ports = append(ports, port)
	}

	// Used to label Service and selector
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       r.Instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.Instance.Name,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Spec.App,
			Namespace: r.Instance.TargetNamespace(),
			Labels:    labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, service, func(runtime.Object) error {
		service.Spec.Ports = ports
		service.Spec.Selector = map[string]string{picchuv1alpha1.LabelApp: r.Instance.Spec.App}
		return nil
	})

	log.Info("Service sync'd", "Op", op)
	return err
}

func (r *ResourceSyncer) SyncDestinationRule() error {
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       r.Instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.Instance.Name,
	}
	appName := r.Instance.Spec.App
	service := fmt.Sprintf("%s.%s.svc.cluster.local", appName, r.Instance.TargetNamespace())
	drule := &istiov1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Spec.App,
			Namespace: r.Instance.TargetNamespace(),
			Labels:    labels,
		},
	}
	subsets := []istiov1alpha3.Subset{}
	for _, incarnation := range r.IncarnationList.Items {
		tag := incarnation.Spec.App.Tag
		subsets = append(subsets, istiov1alpha3.Subset{
			Name:   tag,
			Labels: map[string]string{picchuv1alpha1.LabelTag: tag},
		})
	}
	spec := istiov1alpha3.DestinationRuleSpec{
		Host:    service,
		Subsets: subsets,
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, drule, func(runtime.Object) error {
		drule.Spec = spec
		return nil
	})
	log.Info("DestinationRule sync'd", "Op", op)
	return err
}

func (r *ResourceSyncer) SyncVirtualService() error {
	appName := r.Instance.Spec.App
	defaultDomain := r.Cluster.Spec.DefaultDomain
	serviceHost := fmt.Sprintf("%s.%s.svc.cluster.local", appName, r.Instance.TargetNamespace())
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       appName,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.Instance.Name,
	}
	defaultHost := fmt.Sprintf("%s.%s", r.Instance.TargetNamespace(), defaultDomain)
	meshHost := fmt.Sprintf("%s.%s.svc.cluster.local", appName, r.Instance.TargetNamespace())
	// keep a set of hosts
	hosts := map[string]bool{defaultHost: true, meshHost: true}
	publicGateway := r.Cluster.Spec.Ingresses.Public.Gateway
	privateGateway := r.Cluster.Spec.Ingresses.Private.Gateway
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
			Namespace: r.Instance.TargetNamespace(),
			Labels:    labels,
		},
	}
	http := []istiov1alpha3.HTTPRoute{}

	// NOTE: We are iterating through the incarnations twice, these loops could
	// be combined if we sorted the incarnations by git timestamp at the expense
	// of readability

	// Incarnation specific releases are made for each port on private ingress
	for _, incarnation := range r.IncarnationList.Items {
		if !incarnation.Status.Deployed {
			continue
		}
		tag := incarnation.Spec.App.Tag
		overrideLabel := fmt.Sprintf("pin/%s", appName)
		for _, port := range incarnation.Spec.Ports {
			matches := []istiov1alpha3.HTTPMatchRequest{{
				// mesh traffic from same tag'd service with and test tag
				SourceLabels: map[string]string{
					overrideLabel: tag,
				},
				Port:     uint32(port.Port),
				Gateways: []string{"mesh"},
			}}
			if privateGateway != "" {
				host := fmt.Sprintf("%s-%s", tag, defaultHost)
				hosts[host] = true
				matches = append(matches,
					istiov1alpha3.HTTPMatchRequest{
						// internal traffic with MEDIUM-TAG header
						Headers: map[string]istiocommonv1alpha1.StringMatch{
							"Medium-Tag": {Exact: tag},
						},
						Port:     uint32(port.IngressPort),
						Gateways: []string{privateGateway},
					},
					istiov1alpha3.HTTPMatchRequest{
						// internal traffic with :authority host header
						Authority: &istiocommonv1alpha1.StringMatch{Exact: host},
						Port:      uint32(port.IngressPort),
						Gateways:  []string{privateGateway},
					},
				)
			}

			http = append(http, istiov1alpha3.HTTPRoute{
				Match: matches,
				Route: []istiov1alpha3.DestinationWeight{
					{
						Destination: istiov1alpha3.Destination{
							Host:   serviceHost,
							Port:   istiov1alpha3.PortSelector{Number: uint32(port.Port)},
							Subset: tag,
						},
						Weight: 100,
					},
				},
			})
		}
	}

	// The idea here is we will work through releases from newest to oldest,
	// incrementing their weight if enough time has passed since their last
	// update, and stopping when we reach 100%. This will cause newer releases
	// to take from oldest fnord release.
	var percRemaining uint32 = 100
	// Tracking one route per port number
	releaseRoutes := map[string]istiov1alpha3.HTTPRoute{}
	incarnations, err := r.IncarnationList.SortedReleases()
	log.Info("Got my releases", "Count", len(incarnations))
	if err != nil {
		return err
	}
	count := len(incarnations)
	// setup alerts from latest release
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-rule",
			Namespace: r.Instance.TargetNamespace(),
		},
	}
	if count > 0 {
		latest := incarnations[0]
		if len(latest.Spec.AlertRules) > 0 {
			op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, rule, func(runtime.Object) error {
				rule.Labels = map[string]string{picchuv1alpha1.LabelTag: latest.Spec.App.Tag}
				rule.Spec.Groups = []monitoringv1.RuleGroup{{
					Name:  "picchu.rules",
					Rules: latest.Spec.AlertRules,
				}}
				return nil
			})
			if err != nil {
				return err
			}
			log.Info("Sync'd PrometheusRule", "Op", op)
		} else {
			if err := r.Client.Delete(context.TODO(), rule); err != nil && !errors.IsNotFound(err) {
				return err
			}
			log.Info("Deleted PrometheusRule because no rules are defined in incarnation", "Incarnation.Name", latest.Name)
		}
	} else {
		if err := r.Client.Delete(context.TODO(), rule); err != nil && !errors.IsNotFound(err) {
			return err
		}
		log.Info("Deleted PrometheusRule because there aren't any release incarnation")
	}

	for i, incarnation := range incarnations {
		if incarnation.IsDeleted() || !incarnation.Status.Deployed {
			continue
		}
		releaseStatus := r.Instance.ReleaseStatus(incarnation.Spec.App.Tag)
		oldCurrent := releaseStatus.CurrentPercent
		peak := releaseStatus.PeakPercent
		current := incarnation.CurrentPercentTarget(releaseStatus.LastUpdate, oldCurrent, percRemaining)
		if i+1 == count {
			// TODO(bob): confirm we want to give remaining bandwidth to oldest
			// instance
			// Make sure we use all available bandwidth
			current = percRemaining
		}
		log.Info("CurrentPercentage Update", "Incarnation.Name", incarnation.Name, "Old", oldCurrent, "Current", current)
		if current > peak {
			peak = current
		}
		percRemaining -= current

		if percRemaining == 0 {
			// Mark the oldest released incarnation as "released" and observe latency
			if !releaseStatus.Released {
				rct := incarnation.RevisionCreationTimestamp()
				if !rct.IsZero() {
					rate := incarnation.Spec.Release.Rate
					delay := *rate.DelaySeconds
					increment := rate.Increment
					expected := time.Second * time.Duration(delay*int64(math.Ceil(100.0/float64(increment))))
					latency := time.Since(rct) - expected
					incarnationReleaseLatency.Observe(latency.Seconds())
					releaseStatus.Released = true
				}
			}
		}

		tag := incarnation.Spec.App.Tag

		if oldCurrent != current {
			now := metav1.Now()
			releaseStatus.CurrentPercent = current
			releaseStatus.PeakPercent = peak
			releaseStatus.LastUpdate = &now
			r.Instance.UpdateReleaseStatus(releaseStatus)
		}

		// Wind down retired releases in HPA
		var minReplicas int32 = 1
		var maxReplicas int32 = 1

		if current > 0 {
			for _, port := range incarnation.Spec.Ports {
				portNumber := uint32(port.Port)
				filterHosts := port.Hosts
				gateway := []string{"mesh"}
				switch port.Mode {
				case picchuv1alpha1.PortPublic:
					if publicGateway == "" {
						log.Info("Can't configure publicGateway, undefined in Cluster")
						continue
					}
					gateway = []string{publicGateway}
					portNumber = uint32(port.IngressPort)
				case picchuv1alpha1.PortPrivate:
					if privateGateway == "" {
						log.Info("Can't configure privateGateway, undefined in Cluster")
						continue
					}
					gateway = []string{privateGateway}
					filterHosts = append(filterHosts, defaultHost)
					portNumber = uint32(port.IngressPort)
				}
				releaseRoute, ok := releaseRoutes[port.Name]
				if !ok {
					releaseRoute = istiov1alpha3.HTTPRoute{}
					for _, filterHost := range filterHosts {
						hosts[filterHost] = true
						releaseRoute.Match = append(releaseRoute.Match, istiov1alpha3.HTTPMatchRequest{
							Uri:       &istiocommonv1alpha1.StringMatch{Prefix: "/"},
							Authority: &istiocommonv1alpha1.StringMatch{Exact: filterHost},
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
			minReplicas = *incarnation.Spec.Scale.Min
			maxReplicas = incarnation.Spec.Scale.Max
		}
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tag,
				Namespace: r.Instance.TargetNamespace(),
			},
		}

		op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, hpa, func(runtime.Object) error {
			hpa.Spec.MaxReplicas = maxReplicas
			hpa.Spec.MinReplicas = &minReplicas
			return nil
		})
		if err != nil {
			// This is an indicator that the HPA wasn't found, but *could* be wrong, slightly simpler then
			// unwrapping CreateOrUpdate.
			if hpa.Spec.ScaleTargetRef.Kind == "" {
				log.Info("HPA not found for incarnation", "Incarnation.Name", incarnation.Name)
			} else {
				log.Error(err, "Failed to create/update hpa", "Hpa", hpa, "Op", op)
				return err
			}
		}

		log.Info("HPA sync'd", "Op", op)
	}

	for _, route := range releaseRoutes {
		http = append(http, route)
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, vs, func(runtime.Object) error {
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

	log.Info("VirtualService sync'd", "Op", op)
	if err := utils.UpdateStatus(context.TODO(), r.PicchuClient, r.Instance); err != nil {
		log.Error(err, "Failed to update releasemanager status")
		return err
	}
	return nil
}
