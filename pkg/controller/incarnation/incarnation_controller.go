package incarnation

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log             = logf.Log.WithName("controller_incarnation")
	copyAnnotations = []string{"git-scm.com/ref", "github.com/repository"}
)

type Identity interface {
	GetAPIVersion() string
}

// Add creates a new Incarnation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileIncarnation{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	_, err := builder.SimpleController().
		WithManager(mgr).
		ForType(&picchuv1alpha1.Incarnation{}).
		Build(r)

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIncarnation{}

// ReconcileIncarnation reconciles a Incarnation object
type ReconcileIncarnation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func CopyAnnotations(source map[string]string) map[string]string {
	copy := map[string]string{}

	for _, k := range copyAnnotations {
		if v, ok := source[k]; ok {
			copy[k] = v
		}
	}

	return copy
}

type IncarnationResources struct {
	Incarnation      *picchuv1alpha1.Incarnation
	Cluster          *picchuv1alpha1.Cluster
	SourceSecrets    *corev1.SecretList
	SourceConfigMaps *corev1.ConfigMapList
}

func (b *IncarnationResources) Namespaces() []runtime.Object {
	return []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: b.Incarnation.Spec.App.Name,
				Labels: map[string]string{
					"istio-injection": "enabled",
				},
			},
		},
	}
}

func (b *IncarnationResources) GetConfigObjects() (secrets []*corev1.Secret, configMaps []*corev1.ConfigMap) {
	for _, item := range b.SourceSecrets.Items {
		secrets = append(secrets, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        item.Name,
				Namespace:   b.Incarnation.Spec.App.Name,
				Labels:      item.Labels,
				Annotations: CopyAnnotations(item.Annotations),
			},
			Data: item.Data,
		})
	}
	for _, item := range b.SourceConfigMaps.Items {
		configMaps = append(configMaps, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        item.Name,
				Namespace:   b.Incarnation.Spec.App.Name,
				Labels:      item.Labels,
				Annotations: CopyAnnotations(item.Annotations),
			},
			Data: item.Data,
		})
	}
	return
}

func (b *IncarnationResources) ReplicaSets() []runtime.Object {
	var r []runtime.Object
	secrets, configMaps := b.GetConfigObjects()
	for _, secret := range secrets {
		r = append(r, secret)
	}
	for _, configMap := range configMaps {
		r = append(r, configMap)
	}

	appName := b.Incarnation.Spec.App.Name
	tag := b.Incarnation.Spec.App.Tag
	image := b.Incarnation.Spec.App.Image

	var ports []corev1.ContainerPort
	for _, port := range b.Incarnation.Spec.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          port.Name,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
	}
	var envs []corev1.EnvFromSource
	for _, item := range secrets {
		envs = append(envs, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: item.Name},
			},
		})
	}
	for _, item := range configMaps {
		envs = append(envs, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: item.Name},
			},
		})
	}
	// Used to label ReplicaSet, Container and for Container selector
	labels := map[string]string{
		"medium.build/app": appName,
		"medium.build/tag": tag,
	}
	r = append(r, &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tag,
			Namespace: appName,
			Labels:    labels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &b.Incarnation.Spec.Scale.Min,
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tag,
					Namespace: appName,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							EnvFrom:         envs,
							Image:           image,
							ImagePullPolicy: "IfNotPresent",
							Name:            appName,
							Ports:           ports,
						},
					},
				},
			},
		},
	})
	return r
}

func (b *IncarnationResources) ReplicaSetSelector() types.NamespacedName {
	return types.NamespacedName{
		b.Incarnation.Spec.App.Name,
		b.Incarnation.Spec.App.Tag,
	}
}

func (b *IncarnationResources) GetService() *corev1.Service {
	appName := b.Incarnation.Spec.App.Name
	tag := b.Incarnation.Spec.App.Tag
	var ports []corev1.ServicePort
	for _, port := range b.Incarnation.Spec.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       port.Name,
			Protocol:   port.Protocol,
			Port:       port.Port,
			TargetPort: intstr.FromInt(int(port.ContainerPort)),
		})
	}
	// Used to label Service and selector
	labels := map[string]string{
		"medium.build/app": appName,
		"medium.build/tag": tag,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tag,
			Namespace: appName,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: labels,
		},
	}
}

func (b *IncarnationResources) VirtualServices() []runtime.Object {
	service := b.GetService()
	appName := b.Incarnation.Spec.App.Name
	tag := b.Incarnation.Spec.App.Tag
	defaultDomain := b.Cluster.DefaultDomain()
	labels := map[string]string{
		"medium.build/app": appName,
		"medium.build/tag": tag,
	}
	r := []runtime.Object{service}

	for _, port := range b.Incarnation.Spec.Ports {
		var gateway, host string

		switch mode := port.Mode; mode {
		case picchuv1alpha1.PortPrivate:
			gateway = "private-ingressgateway-cert-merge.istio-system.svc.cluster.local"
			host = fmt.Sprintf("%s-%s.%s", tag, port.Name, defaultDomain)
		case picchuv1alpha1.PortPublic:
			continue
			// gateway = "public-ingressgateway-cert-merge.istio-system.svc.cluster.local"
			// TODO(bob) what is the public host?
			// host = ""
		case picchuv1alpha1.PortLocal:
			continue
		}

		portNumber := uint32(port.Port)
		desthost := fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace)

		r = append(r, &istiov1alpha3.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tag,
				Namespace: appName,
				Labels:    labels,
			},
			Spec: istiov1alpha3.VirtualServiceSpec{
				Hosts:    []string{host},
				Gateways: []string{"mesh", gateway},
				Http: []istiov1alpha3.HTTPRoute{
					istiov1alpha3.HTTPRoute{
						Match: []istiov1alpha3.HTTPMatchRequest{
							istiov1alpha3.HTTPMatchRequest{
								Uri: &istiocommonv1alpha1.StringMatch{
									Prefix: "/",
								},
							},
						},
						Route: []istiov1alpha3.DestinationWeight{
							istiov1alpha3.DestinationWeight{
								Destination: istiov1alpha3.Destination{
									Host: desthost,
									Port: istiov1alpha3.PortSelector{Number: portNumber},
								},
							},
						},
					},
				},
			},
		})
	}

	return r
}

func (b *IncarnationResources) Resources() []runtime.Object {
	resources := b.Namespaces()
	resources = append(resources, b.ReplicaSets()...)
	return append(resources, b.VirtualServices()...)
}

func (b *IncarnationResources) Client(reconciler *ReconcileIncarnation) (client.Client, error) {
	namespacedName := types.NamespacedName{b.Cluster.Namespace, b.Cluster.Name}
	secret := &corev1.Secret{}
	err := reconciler.client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}
	config, err := b.Cluster.Config(secret)
	if err != nil {
		return nil, err
	}
	if config != nil {
		return client.New(config, client.Options{})
	}
	return nil, nil
}

// Reconcile reads that state of the cluster for a Incarnation object and makes changes based on the state read
// and what is in the Incarnation.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIncarnation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Incarnation")

	// Fetch the Incarnation instance
	incarnation := &picchuv1alpha1.Incarnation{}
	err := r.client.Get(context.TODO(), request.NamespacedName, incarnation)
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

	configOpts := client.MatchingLabels(map[string]string{
		"medium.build/tag":              incarnation.Spec.App.Tag,
		"medium.build/app":              incarnation.Spec.App.Name,
		"medium.build/target":           incarnation.Spec.Assignment.Target,
		"config.kbfd.medium.build/type": "environment",
		"kbfd.medium.build/role":        "config",
	})
	clusterOpts := types.NamespacedName{request.Namespace, incarnation.Spec.Assignment.Name}

	cluster := &picchuv1alpha1.Cluster{}
	secrets := &corev1.SecretList{}
	configMaps := &corev1.ConfigMapList{}

	if err = r.client.Get(context.TODO(), clusterOpts, cluster); err != nil {
		return reconcile.Result{}, err
	}
	if err = r.client.List(context.TODO(), configOpts, secrets); err != nil {
		return reconcile.Result{}, err
	}
	if err = r.client.List(context.TODO(), configOpts, configMaps); err != nil {
		return reconcile.Result{}, err
	}

	if !cluster.Spec.Enabled {
		return reconcile.Result{}, nil
	}

	ic := IncarnationResources{
		Incarnation:      incarnation,
		Cluster:          cluster,
		SourceSecrets:    secrets,
		SourceConfigMaps: configMaps,
	}
	remoteClient, err := ic.Client(r)
	if err != nil {
		return reconcile.Result{}, err
	}

	resourceStatus := []picchuv1alpha1.IncarnationResourceStatus{}

	// Create statuses for all resources
	for _, resource := range ic.Resources() {
		status := "unknown"
		apiVersion := ""
		kind := ""

		kinds, _, _ := scheme.Scheme.ObjectKinds(resource)
		if len(kinds) == 1 {
			apiVersion, kind = kinds[0].ToAPIVersionAndKind()
		}

		metadata, _ := apimeta.Accessor(resource)
		namespace := metadata.GetNamespace()
		name := metadata.GetName()

		resourceStatus = append(resourceStatus, picchuv1alpha1.IncarnationResourceStatus{
			ApiVersion: apiVersion,
			Kind:       kind,
			Metadata:   &types.NamespacedName{name, namespace},
			Status:     status,
		})
	}

	// Create All
	err = nil
	for i, resource := range ic.Resources() {
		resourceStatus[i].Status = "creating"
		_, err = controllerutil.CreateOrUpdate(context.TODO(), remoteClient, resource, func(existing runtime.Object) error {
			resourceStatus[i].Status = "created"
			return nil
		})
		if err != nil {
			resourceStatus[i].Status = "error"
			break
		}

	}

	replicaset := &appsv1.ReplicaSet{}
	if e := remoteClient.Get(context.TODO(), ic.ReplicaSetSelector(), replicaset); e != nil {
		reqLogger.Error(e, "Failed to get replicaset")
	}

	health := true
	if replicaset.Status.ReadyReplicas == 0 {
		health = false
	}

	incarnation.Status = picchuv1alpha1.IncarnationStatus{
		Health: picchuv1alpha1.IncarnationHealthStatus{
			// TODO(bob): when we get metrics, evaluate actual health
			Healthy: health,
			Metrics: []picchuv1alpha1.IncarnationHealthMetricStatus{},
		},
		Scale: picchuv1alpha1.IncarnationScaleStatus{
			Current: replicaset.Status.ReadyReplicas,
			Desired: replicaset.Status.Replicas,
		},
		Resources: resourceStatus,
	}
	if e := r.client.Status().Update(context.TODO(), incarnation); e != nil {
		reqLogger.Error(e, "Failed to update Incarnation status")
	}

	return reconcile.Result{}, err
}
