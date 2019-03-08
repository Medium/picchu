package releasemanager

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_releasemanager")

// Add creates a new ReleaseManager Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileReleaseManager{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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

	return reconcile.Result{Requeue: true}, nil
}

type ResourceSyncer struct {
	Instance        *picchuv1alpha1.ReleaseManager
	IncarnationList *picchuv1alpha1.IncarnationList
	Cluster         *picchuv1alpha1.Cluster
	Client          client.Client
	PicchuClient    client.Client
}

func (r *ResourceSyncer) SyncNamespace() error {
	labels := map[string]string{
		"istio-injection": "enabled",
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
		picchuv1alpha1.LabelApp: r.Instance.Spec.App,
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
		service.Spec.Selector = labels
		return nil
	})

	log.Info("Service sync'd", "Op", op)
	return err
}

func (r *ResourceSyncer) SyncDestinationRule() error {
	labels := map[string]string{
		picchuv1alpha1.LabelApp: r.Instance.Spec.App,
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
	labels := map[string]string{picchuv1alpha1.LabelApp: appName}
	defaultHost := fmt.Sprintf("%s.%s", r.Instance.TargetNamespace(), defaultDomain)
	// keep a set of hosts
	hosts := map[string]bool{defaultHost: true}
	// TODO(bob): figure out public and private ingressgateway names and make sure they exist
	// publicIngress := "public-ingressgateway-cert-merge.istio-system.svc.cluster.local"
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
	// as <port.Name>-<tag>-<app>.<defaultDomain>
	for _, incarnation := range r.IncarnationList.Items {
		tag := incarnation.Spec.App.Tag
		for _, port := range incarnation.Spec.Ports {
			matches := []istiov1alpha3.HTTPMatchRequest{{
				// mesh traffic from same tag'd service with and test tag
				SourceLabels: map[string]string{
					picchuv1alpha1.LabelApp:          appName,
					picchuv1alpha1.LabelTag:          tag,
					"picchu.medium.engineering/test": "1",
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
	if err != nil {
		return err
	}
	count := len(incarnations)
	for i, incarnation := range incarnations {
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

		tag := incarnation.Spec.App.Tag

		if oldCurrent != current {
			now := metav1.Now()
			releaseStatus.CurrentPercent = current
			releaseStatus.PeakPercent = peak
			releaseStatus.LastUpdate = &now
			r.Instance.UpdateReleaseStatus(releaseStatus)
		}

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

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, vs, func(runtime.Object) error {
		hostSlice := make([]string, 0, len(hosts))
		for h, _ := range hosts {
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