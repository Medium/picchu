package incarnation

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("controller_incarnation")
)

type Identity interface {
	GetAPIVersion() string
}

// Add creates a new Incarnation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, c utils.Config) error {
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c utils.Config) reconcile.Reconciler {
	return &ReconcileIncarnation{client: mgr.GetClient(), scheme: mgr.GetScheme(), config: c}
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
	config utils.Config
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
	r.scheme.Default(incarnation)

	opts := types.NamespacedName{request.Namespace, incarnation.Spec.Assignment.Name}
	cluster := &picchuv1alpha1.Cluster{}
	if err = r.client.Get(context.TODO(), opts, cluster); err != nil {
		return reconcile.Result{}, err
	}
	r.scheme.Default(cluster)
	remoteClient, err := utils.RemoteClient(r.client, cluster)
	if err != nil {
		reqLogger.Error(err, "Failed to create remote client")
		return reconcile.Result{}, err
	}
	syncer := ResourceSyncer{
		Instance: incarnation,
		Client:   remoteClient,
		Owner:    r,
	}
	if err := syncer.Sync(); err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{Requeue: true}, err
}

func NewIncarnationResourceStatus(resource runtime.Object) picchuv1alpha1.IncarnationResourceStatus {
	status := "created"
	apiVersion := ""
	kind := ""

	kinds, _, _ := scheme.Scheme.ObjectKinds(resource)
	if len(kinds) == 1 {
		apiVersion, kind = kinds[0].ToAPIVersionAndKind()
	}

	metadata, _ := apimeta.Accessor(resource)
	namespace := metadata.GetNamespace()
	name := metadata.GetName()

	return picchuv1alpha1.IncarnationResourceStatus{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata:   &types.NamespacedName{namespace, name},
		Status:     status,
	}
}

type ResourceSyncer struct {
	Instance *picchuv1alpha1.Incarnation
	Client   client.Client
	Owner    *ReconcileIncarnation
}

// SyncConfig syncs ConfigMaps and Secrets
func (r *ResourceSyncer) Sync() error {
	defaultLabels := map[string]string{
		picchuv1alpha1.LabelApp:       r.Instance.Spec.App.Name,
		picchuv1alpha1.LabelTag:       r.Instance.Spec.App.Tag,
		picchuv1alpha1.LabelTarget:    r.Instance.Spec.Assignment.Target,
		picchuv1alpha1.LabelOwnerType: "incarnation",
		picchuv1alpha1.LabelOwnerName: r.Instance.Name,
	}
	var envs []corev1.EnvFromSource
	resourceStatus := []picchuv1alpha1.IncarnationResourceStatus{}
	selector, err := metav1.LabelSelectorAsSelector(r.Instance.Spec.ConfigSelector)
	if err != nil {
		return err
	}
	configOpts := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     r.Instance.Namespace,
	}

	secrets := &corev1.SecretList{}
	configMaps := &corev1.ConfigMapList{}

	if err := r.Owner.client.List(context.TODO(), configOpts, secrets); err != nil {
		return err
	}
	if err := r.Owner.client.List(context.TODO(), configOpts, configMaps); err != nil {
		return err
	}

	// It'd be nice if we could combine the following two blocks.
	for _, item := range secrets.Items {
		remote := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      item.Name,
				Namespace: r.Instance.TargetNamespace(),
			},
		}
		op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, remote, func(runtime.Object) error {
			remote.ObjectMeta.Labels = defaultLabels
			remote.Data = item.Data
			return nil
		})
		status := NewIncarnationResourceStatus(remote)
		if err != nil {
			status.Status = "failed"
			log.Error(err, "Failed to sync secret")
			return err
		}
		resourceStatus = append(resourceStatus, status)
		log.Info("Secret sync'd", "Op", op)
		envs = append(envs, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: remote.Name},
			},
		})
	}
	for _, item := range configMaps.Items {
		remote := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      item.Name,
				Namespace: r.Instance.TargetNamespace(),
			},
		}
		op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, remote, func(runtime.Object) error {
			remote.ObjectMeta.Labels = defaultLabels
			remote.Data = item.Data
			return nil
		})
		status := NewIncarnationResourceStatus(remote)
		if err != nil {
			status.Status = "failed"
			log.Error(err, "Failed to sync configMap")
			return err
		}
		resourceStatus = append(resourceStatus, status)
		log.Info("ConfigMap sync'd", "Op", op)
		envs = append(envs, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: remote.Name},
			},
		})
	}

	appName := r.Instance.Spec.App.Name
	tag := r.Instance.Spec.App.Tag

	var ports []corev1.ContainerPort
	for _, port := range r.Instance.Spec.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          port.Name,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
	}
	// Used for Container label and Container Selector
	labels := map[string]string{
		picchuv1alpha1.LabelApp: appName,
		picchuv1alpha1.LabelTag: tag,
	}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tag,
			Namespace: r.Instance.TargetNamespace(),
			Labels:    defaultLabels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &r.Instance.Spec.Scale.Default,
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tag,
					Namespace: r.Instance.TargetNamespace(),
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							EnvFrom:   envs,
							Image:     r.Instance.Spec.App.Image,
							Name:      r.Instance.Spec.App.Name,
							Ports:     ports,
							Resources: r.Instance.Spec.Resources,
						},
					},
				},
			},
		},
	}

	status := NewIncarnationResourceStatus(replicaSet)
	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, replicaSet, func(runtime.Object) error {
		return nil
	})
	log.Info("ReplicaSet sync'd", "Op", op)
	if err != nil {
		status.Status = "failed"
		log.Error(err, "Failed to sync replicaSet")
		return err
	}
	resourceStatus = append(resourceStatus, status)

	var cpuTarget *int32
	if !r.Instance.Spec.Scale.Resources.CPU.IsZero() {
		// TODO(lyra): clean up
		log.Info("Deprecated resources.cpu field used; use targetCPUUtilizationPercentage")
		cpuTarget = r.Instance.Spec.Scale.TargetCPUUtilizationPercentage
	} else if r.Instance.Spec.Scale.TargetCPUUtilizationPercentage != nil {
		cpuPercentage := int32(r.Instance.Spec.Scale.Resources.CPU.ScaledValue(resource.Milli) / 10)
		cpuTarget = &cpuPercentage
	}

	// TODO(lyra): delete / don't create the HPA if there is no CPU target
	kind := utils.MustGetKind(r.Owner.scheme, replicaSet)
	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tag,
			Namespace: r.Instance.TargetNamespace(),
			Labels:    defaultLabels,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       kind.Kind,
				Name:       replicaSet.Name,
				APIVersion: kind.GroupVersion().String(),
			},
			MinReplicas:                    r.Instance.Spec.Scale.Min,
			MaxReplicas:                    r.Instance.Spec.Scale.Max,
			TargetCPUUtilizationPercentage: cpuTarget,
		},
	}

	status = NewIncarnationResourceStatus(hpa)
	op, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, hpa, func(runtime.Object) error {
		return nil
	})
	log.Info("HPA sync'd", "Op", op)
	if err != nil {
		status.Status = "failed"
		log.Error(err, "Failed to sync hpa")
		return err
	}
	resourceStatus = append(resourceStatus, status)

	inc := &picchuv1alpha1.Incarnation{}
	key, err := client.ObjectKeyFromObject(r.Instance)
	if err != nil {
		return err
	}
	if err := r.Owner.client.Get(context.TODO(), key, inc); err != nil {
		return err
	}

	health := true
	if replicaSet.Status.ReadyReplicas == 0 {
		health = false
	}

	inc.Status.Health.Healthy = health
	inc.Status.Scale.Current = replicaSet.Status.ReadyReplicas
	inc.Status.Scale.Desired = replicaSet.Status.Replicas
	inc.Status.Resources = resourceStatus
	if err = utils.UpdateStatus(context.TODO(), r.Owner.client, inc); err != nil {
		log.Error(err, "Failed to update Incarnation status")
		return err
	}
	return nil
}
