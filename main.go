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

package main

import (
	"flag"
	"os"
	"time"

	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	slo "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	picchumediumengineeringv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	apis "go.medium.engineering/picchu/api/v1alpha1/apis"
	clientgoscheme "go.medium.engineering/picchu/client/scheme"
	"go.medium.engineering/picchu/controllers"
	"go.medium.engineering/picchu/controllers/utils"
	promapi "go.medium.engineering/picchu/prometheus"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(picchumediumengineeringv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	manageRoute53 := flag.Bool("manage-route53", false, "Should picchu manage route53?")
	requeuePeriodSeconds := flag.Int("sync-period-seconds", 15, "Delay between requeues")
	prometheusQueryAddress := flag.String("prometheus-query-address", "", "The (usually thanos) address that picchu should query to SLO alerts")
	prometheusQueryTTL := flag.Duration("prometheus-query-ttl", time.Duration(10)*time.Second, "How long to cache SLO alerts")
	humaneReleasesEnabled := flag.Bool("humane-releases-enabled", true, "Release apps on the humane schedule")
	prometheusEnabled := flag.Bool("prometheus-enabled", true, "Prometheus integration for SLO alerts is enabled")
	serviceLevelsNamespace := flag.String("service-levels-namespace", "service-level-objectives", "The namespace to use when creating ServiceLevel resources in the delivery cluster")
	serviceLevelsFleet := flag.String("service-levels-fleet", "delivery", "The fleet to use when creating ServiceLevel resources")
	concurrentRevisions := flag.Int("concurrent-revisions", 20, "How many concurrent revisions to reconcile")
	concurrentReleaseManagers := flag.Int("concurrent-release-managers", 50, "How many concurrent release managers to reconcile")
	devRoutesServiceHost := flag.String("dev-routes-service-host", "", "Configures the dev routes service host, if cluster dev routes are enabled")
	devRoutesServicePort := flag.Int("dev-routes-service-port", 80, "Configures the dev routes service port, if cluster dev routes are enabled")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	requeuePeriod := time.Duration(*requeuePeriodSeconds) * time.Second
	if !*prometheusEnabled {
		setupLog.Info("SLO alerts will not be respected (--prometheus-enabled=false)")
		*prometheusQueryAddress = ""
	}

	cconfig := utils.Config{
		ManageRoute53:             *manageRoute53,
		HumaneReleasesEnabled:     *humaneReleasesEnabled,
		RequeueAfter:              requeuePeriod,
		PrometheusQueryAddress:    *prometheusQueryAddress,
		PrometheusQueryTTL:        *prometheusQueryTTL,
		ServiceLevelsNamespace:    *serviceLevelsNamespace,
		ServiceLevelsFleet:        *serviceLevelsFleet,
		ConcurrentRevisions:       *concurrentRevisions,
		ConcurrentReleaseManagers: *concurrentReleaseManagers,
		DevRoutesServiceHost:      *devRoutesServiceHost,
		DevRoutesServicePort:      *devRoutesServicePort,
	}

	var api controllers.PromAPI
	var errPromAPI error

	if cconfig.PrometheusQueryAddress != "" {
		api, errPromAPI = promapi.NewAPI(cconfig.PrometheusQueryAddress, cconfig.PrometheusQueryTTL)
	} else {
		api = &controllers.NoopPromAPI{}
	}
	if errPromAPI != nil {
		panic(errPromAPI)
	}

	schemeBuilders := k8sruntime.SchemeBuilder{
		apps.AddToScheme,
		core.AddToScheme,
		autoscaling.AddToScheme,
		picchu.AddToScheme,
		istio.AddToScheme,
		monitoring.AddToScheme,
		slo.AddToScheme,
		wpav1.AddToScheme,
		apis.AddToScheme,
	}

	for _, sch := range []*k8sruntime.Scheme{clientgoscheme.Scheme} {
		if err := picchu.RegisterDefaults(sch); err != nil {
			setupLog.Error(err, "")
			os.Exit(1)
		}

		for _, addToScheme := range schemeBuilders {
			if err := addToScheme(sch); err != nil {
				setupLog.Error(err, "")
				os.Exit(1)
			}
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             clientgoscheme.Scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "de705aec.picchu.medium.engineering",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Cluster"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	if err = (&controllers.ReleaseManagerReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ReleaseManager"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ReleaseManager")
		os.Exit(1)
	}
	if err = (&controllers.ClusterSecretsReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterSecrets"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterSecrets")
		os.Exit(1)
	}
	if err = (&controllers.RevisionReconciler{
		Client:       mgr.GetClient(),
		CustomLogger: ctrl.Log.WithName("controllers").WithName("Revision"),
		Scheme:       mgr.GetScheme(),
		Config:       cconfig,
		PromAPI:      api,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Revision")
		os.Exit(1)
	}
	if err = (&picchumediumengineeringv1alpha1.Revision{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Revision")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
