package releasemanager

import (
	"context"
	"fmt"
	"time"

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

	cluster := &picchuv1alpha1.Cluster{}
	key := client.ObjectKey{request.Namespace, instance.Spec.Cluster}
	if err := r.client.Get(context.TODO(), key, cluster); err != nil {
		return reconcile.Result{}, err
	}
	incarnationList := &picchuv1alpha1.IncarnationList{}
	labelSelector := client.MatchingLabels(map[string]string{
		"medium.build/app":     instance.Spec.App,
		"medium.build/cluster": instance.Spec.Cluster,
	})
	if err := r.client.List(context.TODO(), labelSelector, incarnationList); err != nil {
		return reconcile.Result{}, err
	}

	remoteClient, err := utils.RemoteClient(r.client, cluster)
	if err != nil {
		reqLogger.Error(err, "Failed to create remote client", "Cluster.Key", key)
		return reconcile.Result{}, err
	}
	reqLogger.Info("Created remote client", "Cluster.Key", key)
	manager := ReleaseManager{
		Instance:        instance,
		IncarnationList: incarnationList,
		Cluster:         cluster,
		Client:          remoteClient,
		PicchuClient:    r.client,
	}
	if err := manager.SyncNamespace(); err != nil {
		return reconcile.Result{}, err
	}
	if err := manager.SyncService(); err != nil {
		return reconcile.Result{}, err
	}
	if err := manager.SyncDestinationRule(); err != nil {
		return reconcile.Result{}, err
	}
	if err := manager.SyncVirtualService(); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

type ReleaseManager struct {
	Instance        *picchuv1alpha1.ReleaseManager
	IncarnationList *picchuv1alpha1.IncarnationList
	Cluster         *picchuv1alpha1.Cluster
	Client          client.Client
	PicchuClient    client.Client
}

func (r *ReleaseManager) SyncNamespace() error {
	labels := map[string]string{
		"istio-injection": "enabled",
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   r.Instance.Spec.App,
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

func (r *ReleaseManager) SyncService() error {
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
	ports := make([]corev1.ServicePort, len(portMap))
	i := 0
	for _, port := range portMap {
		ports[i] = port
		i++
	}

	// Used to label Service and selector
	labels := map[string]string{
		"medium.build/app": r.Instance.Spec.App,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Spec.App,
			Namespace: r.Instance.Spec.App,
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

func (r *ReleaseManager) SyncDestinationRule() error {
	labels := map[string]string{
		"medium.build/app": r.Instance.Spec.App,
	}
	appName := r.Instance.Spec.App
	service := fmt.Sprintf("%s.%s.svc.cluster.local", appName, appName)
	drule := &istiov1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Spec.App,
			Namespace: r.Instance.Spec.App,
			Labels:    labels,
		},
		Spec: istiov1alpha3.DestinationRuleSpec{
			Host: service,
		},
	}
	subsets := []istiov1alpha3.Subset{}
	for _, incarnation := range r.IncarnationList.Items {
		tag := incarnation.Spec.App.Tag
		subsets = append(subsets, istiov1alpha3.Subset{
			Name:   tag,
			Labels: map[string]string{"medium.build/tag": tag},
		})
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, drule, func(runtime.Object) error {
		drule.Spec.Subsets = subsets
		return nil
	})
	log.Info("DestinationRule sync'd", "Op", op)
	return err
}

func (r *ReleaseManager) SyncVirtualService() error {
	appName := r.Instance.Spec.App
	defaultDomain := r.Cluster.DefaultDomain()
	serviceHost := fmt.Sprintf("%s.%s.svc.cluster.local", appName, appName)
	labels := map[string]string{"medium.build/app": appName}
	defaultHost := fmt.Sprintf("%s.%s", appName, defaultDomain)
	hosts := []string{defaultHost}
	// TODO(bob): figure out public and private ingressgateway names and make sure they exist
	// publicIngress := "public-ingressgateway-cert-merge.istio-system.svc.cluster.local"
	privateIngress := "private-ingressgateway-cert-merge.istio-system.svc.cluster.local"
	gateways := []string{
		"mesh",
		// publicIngress,
		privateIngress,
	}
	vs := &istiov1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: appName,
			Labels:    labels,
		},
	}
	http := []istiov1alpha3.HTTPRoute{}

	// NOTE: We are iterating through the incarnations twice, these loops could
	// be combined if we sorted the incarnations by git timestamp at the expense
	// of readability

	// Incarnation specific releases are made for each port on private ingress
	// as <port.Name>-<tag>-<app>.<fleet>.medm.io
	for _, incarnation := range r.IncarnationList.Items {
		tag := incarnation.Spec.App.Tag
		host := fmt.Sprintf("%s-%s.%s", tag, appName, defaultDomain)
		hosts = append(hosts, host)

		for _, port := range incarnation.Spec.Ports {
			route := istiov1alpha3.HTTPRoute{
				Match: []istiov1alpha3.HTTPMatchRequest{
					{
						// internal traffic with MEDIUM-TAG header
						Headers: map[string]istiocommonv1alpha1.StringMatch{
							"Medium-Tag": {Exact: tag},
						},
						Port:     uint32(port.Port),
						Gateways: []string{privateIngress},
					},
					{
						// internal traffic with :authority host header
						Authority: &istiocommonv1alpha1.StringMatch{Exact: host},
						Port:      uint32(port.Port),
						Gateways:  []string{privateIngress},
					},
					{
						// mesh traffic from same tag'd service with and test tag
						SourceLabels: map[string]string{
							"medium.build/app": appName,
							"medium.build/tag": tag,
							// TODO(bob): formalize what tag should be routed
							"medium.build/test": "1",
						},
						Port:     uint32(port.Port),
						Gateways: []string{"mesh"},
					},
				},
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
			}
			http = append(http, route)
		}
	}

	// The idea here is we will work through releases from newest to oldest,
	// incrementing their weight if enough time has passed since their last
	// update, and stopping when we reach 100%. This will cause newer releases
	// to take from oldest fnord release.
	percRemaining := int32(100)
	// Tracking one route per port number
	releaseRoutes := map[uint32]istiov1alpha3.HTTPRoute{}
	for _, incarnation := range r.IncarnationList.SortedReleases() {
		oldCurrent := incarnation.Status.Release.CurrentPercent
		peak := incarnation.Status.Release.PeakPercent
		log.Info("Finding target perc", "incarnation.Name", incarnation.Name, "percRemaining", percRemaining, "gitTimestamp", incarnation.GitTimestamp(), "annotations", incarnation.Annotations)
		current := incarnation.CurrentPercentTarget(percRemaining)
		if current > percRemaining {
			log.Info("Impossible")
			return fmt.Errorf("WTF")
		}
		if current > peak {
			peak = current
		}
		percRemaining = percRemaining - current
		log.Info("Found release incarnation", "incarnation.Name", incarnation.Name, "targetPercent", current)

		tag := incarnation.Spec.App.Tag

		if oldCurrent != current {
			incarnation.Status.Release.CurrentPercent = current
			incarnation.Status.Release.PeakPercent = peak
			incarnation.Status.Release.LastUpdate = time.Now().Format(time.RFC3339)
			if err := r.PicchuClient.Status().Update(context.TODO(), &incarnation); err != nil {
				return err
			}
		}

		if current > 0 {
			for _, port := range incarnation.Spec.Ports {
				portNumber := uint32(port.Port)
				releaseRoute, ok := releaseRoutes[portNumber]
				if !ok {
					releaseRoute = istiov1alpha3.HTTPRoute{
						Match: []istiov1alpha3.HTTPMatchRequest{{
							Uri:       &istiocommonv1alpha1.StringMatch{Prefix: "/"},
							Authority: &istiocommonv1alpha1.StringMatch{Exact: defaultHost},
							Port:      uint32(port.Port),
							Gateways:  []string{privateIngress},
						}},
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
				releaseRoutes[portNumber] = releaseRoute
			}
		}
	}

	for _, route := range releaseRoutes {
		log.Info("Adding release route")
		http = append(http, route)
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, vs, func(runtime.Object) error {
		vs.Spec.Hosts = hosts
		vs.Spec.Gateways = gateways
		vs.Spec.Http = http
		return nil
	})
	log.Info("VirtualService sync'd", "Op", op)
	return err
}
