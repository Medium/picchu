package incarnation

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
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
		Instance:     incarnation,
		Client:       remoteClient,
		PicchuClient: r.client,
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
	Instance     *picchuv1alpha1.Incarnation
	Client       client.Client
	PicchuClient client.Client
}

// SyncConfig syncs ConfigMaps and Secrets
func (r *ResourceSyncer) Sync() error {
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

	if err := r.PicchuClient.List(context.TODO(), configOpts, secrets); err != nil {
		return err
	}
	if err := r.PicchuClient.List(context.TODO(), configOpts, configMaps); err != nil {
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
			remote.ObjectMeta.Labels = item.Labels
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
			remote.ObjectMeta.Labels = item.Labels
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
	// Used to label ReplicaSet, Container and for Container selector
	labels := map[string]string{
		picchuv1alpha1.LabelApp: appName,
		picchuv1alpha1.LabelTag: tag,
	}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tag,
			Namespace: r.Instance.TargetNamespace(),
		},
	}

	status := NewIncarnationResourceStatus(replicaSet)
	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, replicaSet, func(runtime.Object) error {
		replicaSet.ObjectMeta.Labels = labels
		replicaSet.Spec = appsv1.ReplicaSetSpec{
			Replicas: &r.Instance.Spec.Scale.Min,
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
							EnvFrom: envs,
							Image:   r.Instance.Spec.App.Image,
							Name:    r.Instance.Spec.App.Name,
							Ports:   ports,
						},
					},
				},
			},
		}
		return nil
	})
	log.Info("ReplicaSet sync'd", "Op", op)
	if err != nil {
		status.Status = "failed"
		log.Error(err, "Failed to sync replicaSet")
		return err
	}
	resourceStatus = append(resourceStatus, status)

	health := true
	if replicaSet.Status.ReadyReplicas == 0 {
		health = false
	}

	r.Instance.Status.Health.Healthy = health
	r.Instance.Status.Scale.Current = replicaSet.Status.ReadyReplicas
	r.Instance.Status.Scale.Desired = replicaSet.Status.Replicas
	r.Instance.Status.Resources = resourceStatus
	if err = utils.UpdateStatus(context.TODO(), r.PicchuClient, r.Instance); err != nil {
		log.Error(err, "Failed to update Incarnation status")
		return err
	}
	return nil
}
