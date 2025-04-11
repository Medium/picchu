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

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"
	"strings"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.medium.engineering/picchu/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// TODO(bob): Add these to Revision type
	AcceptancePercentage uint32 = 50
)

var (
	clog = logf.Log.WithName("controller_revision")

	revisionFailedGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "picchu_revision_failed",
		Help: "track failed revisions",
	}, []string{"app", "tag"})
	mirrorFailureCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "picchu_mirror_failure_counter",
		Help: "Record picchu mirror failures",
	}, []string{"app", "mirror"})
)

// +kubebuilder:rbac:groups=picchu.medium.engineering,resources=revisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=picchu.medium.engineering,resources=revisions/status,verbs=get;update;patch

func (r *RevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.Config.ConcurrentRevisions}).
		For(&picchuv1alpha1.Revision{}).
		Complete(r)
}

type PromAPI interface {
	IsRevisionTriggered(ctx context.Context, name, tag string, withCanary bool) (bool, []string, error)
}

type NoopPromAPI struct{}

func (n *NoopPromAPI) IsRevisionTriggered(ctx context.Context, name, tag string, withCanary bool) (bool, []string, error) {
	return false, nil, nil
}

type DatadogMonitorAPI interface {
	IsRevisionTriggered(ctx context.Context, name, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, []string, error)
}

type NoopDatadogMonitorAPI struct{}

func (n *NoopDatadogMonitorAPI) IsRevisionTriggered(ctx context.Context, name, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, []string, error) {
	return false, nil, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, c utils.Config) error {
	_, err := builder.ControllerManagedBy(mgr).
		For(&picchuv1alpha1.Revision{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: c.ConcurrentRevisions}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(_ event.UpdateEvent) bool { return false },
		}).
		Build(r)

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &RevisionReconciler{}

// RevisionReconciler reconciles a Revision object
type RevisionReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client            client.Client
	Scheme            *runtime.Scheme
	Config            utils.Config
	PromAPI           PromAPI
	DatadogMonitorAPI DatadogMonitorAPI
	CustomLogger      logr.Logger
}

// Reconcile reads that state of the cluster for a Revision object and makes changes based on the state read
// and what is in the Revision.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *RevisionReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	traceID := uuid.New().String()
	reqLogger := clog.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "Trace", traceID)
	if r.CustomLogger != (logr.Logger{}) {
		reqLogger = r.CustomLogger
	}

	// Fetch the Revision instance

	instance := &picchuv1alpha1.Revision{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return r.NoRequeue(reqLogger, nil)
		}
		// Error reading the object - requeue the request.
		return r.Requeue(reqLogger, err)
	}
	r.Scheme.Default(instance)
	log := reqLogger.WithValues("App", instance.Spec.App.Name, "Tag", instance.Spec.App.Tag)

	mirrors := &picchuv1alpha1.MirrorList{}
	err = r.Client.List(ctx, mirrors)
	if err != nil {
		return r.Requeue(log, nil)
	}

	if err = r.LabelWithAppAndFleets(log, instance); err != nil {
		return r.Requeue(log, err)
	}

	promLabels := prometheus.Labels{
		"app": instance.Spec.App.Name,
		"tag": instance.Spec.App.Tag,
	}

	deleted, err := r.deleteIfMarked(log, instance)
	if err != nil {
		return r.Requeue(log, err)
	}

	if deleted {
		return r.Requeue(log, nil)
	}

	status, err := r.syncReleaseManager(log, instance)
	if err != nil {
		return r.Requeue(log, err)
	}

	isDeployed := false
	for i := range status.Targets {
		if status.Targets[i].Scale.Current+status.Targets[i].Scale.Desired > 0 {
			isDeployed = true
		}
	}
	if isDeployed && !instance.Spec.DisableMirroring {
		for i := range mirrors.Items {
			mirror := mirrors.Items[i]
			err = r.mirrorRevision(ctx, log, &mirror, instance)
			if err != nil {
				log.Error(err, "Failed to mirror revision", "Mirror", mirror.Spec.ClusterName)
				mLabels := prometheus.Labels{
					"app":    instance.Spec.App.Name,
					"mirror": mirror.Spec.ClusterName,
				}
				mirrorFailureCounter.With(mLabels).Inc()
			}
		}
	}

	triggered, alarms, err := r.PromAPI.IsRevisionTriggered(context.TODO(), instance.Spec.App.Name, instance.Spec.App.Tag, instance.Spec.CanaryWithSLIRules)

	if err != nil {
		return r.Requeue(log, err)
	}

	ddog_triggered := false
	var ddog_err error
	var monitors []string

	// testing echo canary phase with datadog api call
	if instance.Spec.App.Name == "echo" {
		var datadogSLOs []*picchuv1alpha1.DatadogSLO

		// grab datadog slos from production target
		for _, t := range instance.Spec.Targets {
			if strings.Contains(t.Name, "production") && t.DatadogSLOsEnabled {
				datadogSLOs = t.DatadogSLOs
			}
		}

		// determine if production target is in the canary phase
		for _, statusTarget := range status.Targets {
			if strings.Contains(statusTarget.Name, "production") {
				if statusTarget.State == "canarying" {
					log.Info("echo found canary state for production target")
					ddog_triggered, monitors, ddog_err = r.DatadogMonitorAPI.IsRevisionTriggered(context.TODO(), instance.Spec.App.Name, instance.Spec.App.Tag, datadogSLOs)
					if ddog_err != nil {
						return r.Requeue(log, ddog_err)
					}
				}
			}
		}
	}

	var revisionFailing bool
	var revisionFailingReason string
	for _, statusTarget := range status.Targets {
		if statusTarget.State == "timingout" {
			revisionFailing = true
			revisionFailingReason = "test timing out"
		}
	}

	// this will be true if there are datadog slos defined under the production target
	if !revisionFailing && ddog_triggered && !instance.Spec.IgnoreSLOs && instance.Spec.App.Name == "echo" {
		log.Info("Revision triggered", "datadog canary monitors", monitors)
		targetStatusMap := map[string]*picchuv1alpha1.RevisionTargetStatus{}
		for i := range status.Targets {
			targetStatusMap[status.Targets[i].Name] = &status.Targets[i]
		}

		for _, revisionTarget := range instance.Spec.Targets {
			// if production target continue with datadog monitor Violation
			if revisionTarget.AcceptanceTarget || strings.Contains(revisionTarget.Name, "production") {
				targetStatus := targetStatusMap[revisionTarget.Name]
				if targetStatus == nil {
					continue
				}
				if targetStatus.Release.PeakPercent < AcceptancePercentage {
					revisionFailing = true
					revisionFailingReason = "datadog canary monitor violation"

					rm, _, err := r.getReleaseManager(log, &revisionTarget, instance)
					if err != nil {
						log.Error(err, "could not getReleaseManager", "revisionTarget", revisionTarget.Name)
						break
					}
					if rm == nil {
						log.Info("missing ReleaseManager", "revisionTarget", revisionTarget.Name)
						break
					}
					revisionStatus := rm.RevisionStatus(instance.Spec.App.Tag)
					revisionStatus.TriggeredDatadogMonitors = monitors
					rm.UpdateRevisionStatus(revisionStatus)
					if err := utils.UpdateStatus(ctx, r.Client, rm); err != nil {
						log.Error(err, "Could not save datadog monitors to RevisionStatus", "alarms", alarms)
					}
				}
				break
			}
		}
	}

	if !revisionFailing && triggered && !instance.Spec.IgnoreSLOs {
		log.Info("Revision triggered", "alarms", alarms)
		targetStatusMap := map[string]*picchuv1alpha1.RevisionTargetStatus{}
		for i := range status.Targets {
			targetStatusMap[status.Targets[i].Name] = &status.Targets[i]
		}

		for _, revisionTarget := range instance.Spec.Targets {
			// if production target continue with SLO Violation
			if revisionTarget.AcceptanceTarget || strings.Contains(revisionTarget.Name, "production") {
				targetStatus := targetStatusMap[revisionTarget.Name]
				if targetStatus == nil {
					continue
				}
				if targetStatus.Release.PeakPercent < AcceptancePercentage {
					revisionFailing = true
					revisionFailingReason = "SLO violation"

					rm, _, err := r.getReleaseManager(log, &revisionTarget, instance)
					if err != nil {
						log.Error(err, "could not getReleaseManager", "revisionTarget", revisionTarget.Name)
						break
					}
					if rm == nil {
						log.Info("missing ReleaseManager", "revisionTarget", revisionTarget.Name)
						break
					}
					revisionStatus := rm.RevisionStatus(instance.Spec.App.Tag)
					revisionStatus.TriggeredAlarms = alarms
					rm.UpdateRevisionStatus(revisionStatus)
					if err := utils.UpdateStatus(ctx, r.Client, rm); err != nil {
						log.Error(err, "Could not save alarms to RevisionStatus", "alarms", alarms)
					}
				}
				break
			}
		}
	}

	if revisionFailing {
		op, err := controllerutil.CreateOrUpdate(ctx, r.Client, instance, func() error {
			instance.Fail()
			return nil
		})
		if err != nil {
			return r.Requeue(log, err)
		}
		log.Info("Set Revision State to failed", "Op", op, "Reason", revisionFailingReason)
		revisionFailedGauge.With(promLabels).Set(float64(1))
	} else {
		revisionFailedGauge.With(promLabels).Set(float64(0))
	}

	instance.Status = status
	if err = r.Client.Status().Update(context.TODO(), instance); err != nil {
		return r.Requeue(log, err)
	}

	return r.Requeue(log, nil)
}

func (r *RevisionReconciler) Requeue(log logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		log.Error(err, "Reconcile resulted in error")
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: r.Config.RequeueAfter}, nil
}

func (r *RevisionReconciler) NoRequeue(log logr.Logger, err error) (reconcile.Result, error) {
	if err != nil {
		log.Error(err, "Reconcile resulted in error")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *RevisionReconciler) LabelWithAppAndFleets(log logr.Logger, revision *picchuv1alpha1.Revision) error {
	var fleetLabels []string
	updated := false
	for _, target := range revision.Spec.Targets {
		name := fmt.Sprintf("%s%s", picchuv1alpha1.LabelFleetPrefix, target.Fleet)
		fleetLabels = append(fleetLabels, name)
		if _, ok := revision.Labels[name]; !ok {
			revision.Labels[name] = ""
			updated = true
		}
	}
	for name := range revision.Labels {
		if strings.HasPrefix(name, picchuv1alpha1.LabelFleetPrefix) {
			found := false
			for _, expected := range fleetLabels {
				if name == expected {
					found = true
					break
				}
			}
			if !found {
				delete(revision.Labels, name)
				updated = true
			}
		}
	}

	if _, ok := revision.Labels[picchuv1alpha1.LabelApp]; !ok {
		revision.Labels[picchuv1alpha1.LabelApp] = revision.Spec.App.Name
		updated = true
	}

	if updated {
		return r.Client.Update(context.TODO(), revision)
	}
	return nil
}

func (r *RevisionReconciler) getReleaseManager(
	log logr.Logger,
	target *picchuv1alpha1.RevisionTarget,
	revision *picchuv1alpha1.Revision,
) (rm *picchuv1alpha1.ReleaseManager, lbls map[string]string, err error) {
	rms := &picchuv1alpha1.ReleaseManagerList{}
	lbls = map[string]string{
		picchuv1alpha1.LabelTarget: target.Name,
		picchuv1alpha1.LabelFleet:  target.Fleet,
		picchuv1alpha1.LabelApp:    revision.Spec.App.Name,
	}
	opts := &client.ListOptions{
		Namespace:     revision.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	r.Client.List(context.TODO(), rms, opts)
	if len(rms.Items) > 1 {
		panic(fmt.Sprintf("Too many ReleaseManagers matching %#v", lbls))
	}
	if len(rms.Items) == 1 {
		rm = &rms.Items[0]
	}

	return
}

func (r *RevisionReconciler) getOrCreateReleaseManager(
	log logr.Logger,
	target *picchuv1alpha1.RevisionTarget,
	revision *picchuv1alpha1.Revision,
) (rm *picchuv1alpha1.ReleaseManager, err error) {
	var lbls map[string]string
	if rm, lbls, err = r.getReleaseManager(log, target, revision); err == nil && rm != nil {
		return
	}

	rm = &picchuv1alpha1.ReleaseManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", revision.Spec.App.Name, target.Name),
			Namespace: revision.Namespace,
			Labels:    lbls,
			Finalizers: []string{
				picchuv1alpha1.FinalizerReleaseManager,
			},
		},
		Spec: picchuv1alpha1.ReleaseManagerSpec{
			Fleet:  target.Fleet,
			App:    revision.Spec.App.Name,
			Target: target.Name,
		},
	}
	if err := r.Client.Create(context.TODO(), rm); err != nil {
		log.Error(err, "Failed to sync releaseManager")
		return nil, err
	}
	log.Info("ReleaseManager sync'd", "Type", "ReleaseManager", "Op", "created", "Content", rm, "Audit", true)

	return
}

func (r *RevisionReconciler) syncReleaseManager(log logr.Logger, revision *picchuv1alpha1.Revision) (picchuv1alpha1.RevisionStatus, error) {
	// Sync releasemanagers
	rstatus := picchuv1alpha1.RevisionStatus{}
	for _, target := range revision.Spec.Targets {
		status := picchuv1alpha1.RevisionTargetStatus{Name: target.Name}
		rm, err := r.getOrCreateReleaseManager(log, &target, revision)
		if err != nil {
			return rstatus, err
		}
		status.AddReleaseManagerStatus(*rm.RevisionStatus(revision.Spec.App.Tag))
		rstatus.AddTarget(status)
	}

	return rstatus, nil
}

func (r *RevisionReconciler) deleteIfMarked(log logr.Logger, revision *picchuv1alpha1.Revision) (bool, error) {
	for _, target := range revision.Spec.Targets {
		label := fmt.Sprintf("%s%s", picchuv1alpha1.LabelTargetDeletablePrefix, target.Name)
		if val, ok := revision.Labels[label]; !ok && val != "true" {
			return false, nil
		}
	}

	log.Info("Deleting revision marked for deletion in all targets")
	err := r.Client.Delete(context.TODO(), revision)
	if err != nil && !errors.IsNotFound(err) {
		return true, err
	}
	return true, nil
}

func (r *RevisionReconciler) mirrorRevision(
	ctx context.Context,
	log logr.Logger,
	mirror *picchuv1alpha1.Mirror,
	revision *picchuv1alpha1.Revision,
) error {
	log.Info("Mirroring revision", "Mirror", mirror.Spec.ClusterName)
	cluster := &picchuv1alpha1.Cluster{}
	key := types.NamespacedName{Namespace: revision.Namespace, Name: mirror.Spec.ClusterName}
	if err := r.Client.Get(ctx, key, cluster); err != nil {
		return err
	}
	remoteClient, err := utils.RemoteClient(ctx, log, r.Client, cluster)
	if err != nil {
		log.Error(err, "Failed to initialize remote client")
		return err
	}
	for i := range revision.Spec.Targets {
		target := revision.Spec.Targets[i]
		log.Info("Syncing target config", "Target", target.Name)
		selector, err := metav1.LabelSelectorAsSelector(target.ConfigSelector)
		if err != nil {
			log.Error(err, "Failed to sync target config")
			return err
		}
		opts := &client.ListOptions{
			LabelSelector: selector,
			Namespace:     revision.Namespace,
		}
		configMapList := &corev1.ConfigMapList{}
		log.Info("Listing configMaps")
		if err := r.Client.List(ctx, configMapList, opts); err != nil {
			log.Error(err, "Failed to list target configMaps")
			return err
		}
		if err := r.copyConfigMapList(ctx, log, remoteClient, configMapList); err != nil {
			log.Error(err, "Failed to sync target configMaps")
			return err
		}
		secretList := &corev1.SecretList{}
		log.Info("Listing secrets")
		if err := r.Client.List(ctx, secretList, opts); err != nil {
			log.Error(err, "Failed to list target secrets")
			return err
		}
		if err := r.copySecretList(ctx, log, remoteClient, secretList); err != nil {
			log.Error(err, "Failed to sync target secrets")
			return err
		}
		externalSecretList := &es.ExternalSecretList{}
		log.Info("Listing ExternalSecrets")
		if err := r.Client.List(ctx, externalSecretList, opts); err != nil {
			log.Error(err, "Failed to list target ExternalSecrets")
			return err
		}
		if err := r.copyExternalSecretList(ctx, log, remoteClient, externalSecretList); err != nil {
			log.Error(err, "Failed to sync target ExternalSecrets")
			return err
		}
	}

	log.Info("Copying additional configs", "Count", len(mirror.Spec.AdditionalConfigSelectors))

	for i := range mirror.Spec.AdditionalConfigSelectors {
		configSelector := mirror.Spec.AdditionalConfigSelectors[i]
		log.Info("Syncing additional configs", "AdditionalConfigSelector", configSelector)
		labelSelector := configSelector.LabelSelector
		if labelSelector == nil {
			labelSelector = &metav1.LabelSelector{
				MatchLabels:      map[string]string{},
				MatchExpressions: nil,
			}
		} else if labelSelector.MatchLabels == nil {
			labelSelector.MatchLabels = map[string]string{}
		}
		labelSelector.MatchLabels[configSelector.AppLabelName] = revision.Spec.App.Name
		labelSelector.MatchLabels[configSelector.TagLabelName] = revision.Spec.App.Tag
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			log.Error(err, "Failed to create Selector for additionalConfig")
			return err
		}
		opts := &client.ListOptions{
			LabelSelector: selector,
			Namespace:     configSelector.Namespace,
		}
		configMapList := &corev1.ConfigMapList{}
		if err := r.Client.List(ctx, configMapList, opts); err != nil {
			log.Error(err, "Failed to list additionalConfigSelector configMaps")
			return err
		}
		if err := r.copyConfigMapList(ctx, log, remoteClient, configMapList); err != nil {
			log.Error(err, "Failed to sycn additionalConfigSelector configMaps")
			return err
		}

		secretList := &corev1.SecretList{}
		if err := r.Client.List(ctx, secretList, opts); err != nil {
			log.Error(err, "Failed to list additionalConfigSelector secrets")
			return err
		}
		if err := r.copySecretList(ctx, log, remoteClient, secretList); err != nil {
			log.Error(err, "Failed to sync additionalConfigSelector secrets")
			return err
		}
	}

	revCopy := &picchuv1alpha1.Revision{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: revision.Annotations,
			Name:        revision.Name,
			Namespace:   revision.Namespace,
			Labels:      revision.Labels,
		},
		Spec: revision.DeepCopy().Spec,
	}
	for i := range revCopy.Spec.Targets {
		revCopy.Spec.Targets[i].ExternalTest.Enabled = false
	}
	log.Info("Syncing revision", "revision", revCopy)
	_, err = controllerutil.CreateOrUpdate(ctx, remoteClient, revCopy, func() error {
		revCopy.Spec = revision.Spec
		return nil
	})
	if err != nil {
		log.Error(err, "Failed to sync revision")
	}
	return err
}

func (r *RevisionReconciler) copyConfigMapList(
	ctx context.Context,
	log logr.Logger,
	remoteClient client.Client,
	configMapList *corev1.ConfigMapList,
) error {
	log.Info("Syncing ConfigMaps", "ConfigMap", configMapList.Items)
	for i := range configMapList.Items {
		orig := configMapList.Items[i]
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: orig.Annotations,
				Name:        orig.Name,
				Namespace:   orig.Namespace,
				Labels:      orig.Labels,
			},
			Data: orig.Data,
		}
		_, err := controllerutil.CreateOrUpdate(ctx, remoteClient, configMap, func() error {
			configMap.Data = configMapList.Items[i].Data
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RevisionReconciler) copySecretList(
	ctx context.Context,
	log logr.Logger,
	remoteClient client.Client,
	secretList *corev1.SecretList,
) error {
	log.Info("Syncing Secrets", "Secrets", secretList.Items)
	for i := range secretList.Items {
		orig := secretList.Items[i]
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: orig.Annotations,
				Name:        orig.Name,
				Namespace:   orig.Namespace,
				Labels:      orig.Labels,
			},
			Type: orig.Type,
			Data: orig.Data,
		}
		_, err := controllerutil.CreateOrUpdate(ctx, remoteClient, secret, func() error {
			secret.Data = orig.Data
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RevisionReconciler) copyExternalSecretList(
	ctx context.Context,
	log logr.Logger,
	remoteClient client.Client,
	externalSecretList *es.ExternalSecretList,
) error {
	log.Info("Syncing ExternalSecrets", "ExternalSecrets", externalSecretList.Items)
	for i := range externalSecretList.Items {
		orig := externalSecretList.Items[i]
		externalSecret := &es.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: orig.Annotations,
				Name:        orig.Name,
				Namespace:   orig.Namespace,
				Labels:      orig.Labels,
			},
			Spec: orig.Spec,
		}
		_, err := controllerutil.CreateOrUpdate(ctx, remoteClient, externalSecret, func() error {
			externalSecret.Spec = orig.Spec
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
