package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/leader"

	//"github.com/operator-framework/operator-sdk/internal/log/zap"

	//"github.com/operator-framework/operator-sdk/pkg/metrics"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	slo "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/spf13/pflag"
	"go.medium.engineering/picchu/pkg/apis"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/client/scheme"
	"go.medium.engineering/picchu/pkg/controller"
	"go.medium.engineering/picchu/pkg/controller/utils"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	core "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)
var log = logf.Log.WithName("cmd")

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(&pflag.FlagSet{})

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	manageRoute53 := pflag.Bool("manage-route53", false, "Should picchu manage route53?")
	requeuePeriodSeconds := pflag.Int("sync-period-seconds", 15, "Delay between requeues")
	prometheusQueryAddress := pflag.String("prometheus-query-address", "", "The (usually thanos) address that picchu should query to SLO alerts")
	prometheusQueryTTL := pflag.Duration("prometheus-query-ttl", time.Duration(10)*time.Second, "How long to cache SLO alerts")
	humaneReleasesEnabled := pflag.Bool("humane-releases-enabled", true, "Release apps on the humane schedule")
	prometheusEnabled := pflag.Bool("prometheus-enabled", true, "Prometheus integration for SLO alerts is enabled")
	serviceLevelsNamespace := pflag.String("service-levels-namespace", "service-levels", "The namespace to use when creating ServiceLevel resources in the delivery cluster")
	serviceLevelsFleet := pflag.String("service-levels-fleet", "delivery", "The fleet to use when creating ServiceLevel resources")
	concurrentRevisions := pflag.Int("concurrent-revisions", 20, "How many concurrent revisions to reconcile")
	concurrentReleaseManagers := pflag.Int("concurrent-release-managers", 50, "How many concurrent release managers to reconcile")
	devRoutesServiceHost := pflag.String("dev-routes-service-host", "", "Configures the dev routes service host, if cluster dev routes are enabled")
	devRoutesServicePort := pflag.Int("dev-routes-service-port", 80, "Configures the dev routes service port, if cluster dev routes are enabled")

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(logr.Logger{})

	if !*humaneReleasesEnabled {
		log.Info("New revisions with the default (humane) schedule will not be released (--humane-releases-enabled=false)")
	}
	if !*prometheusEnabled {
		log.Info("SLO alerts will not be respected (--prometheus-enabled=false)")
		*prometheusQueryAddress = ""
	}

	namespace, err := utils.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "picchu-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	requeuePeriod := time.Duration(*requeuePeriodSeconds) * time.Second

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	schemeBuilders := k8sruntime.SchemeBuilder{
		apps.AddToScheme,
		core.AddToScheme,
		apis.AddToScheme,
		autoscaling.AddToScheme,
		picchu.AddToScheme,
		istio.AddToScheme,
		monitoring.AddToScheme,
		slo.AddToScheme,
		wpav1.AddToScheme,
	}

	for _, sch := range []*k8sruntime.Scheme{mgr.GetScheme(), scheme.Scheme} {
		if err := picchu.RegisterDefaults(sch); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}

		for _, addToScheme := range schemeBuilders {
			if err := addToScheme(sch); err != nil {
				log.Error(err, "")
				os.Exit(1)
			}
		}
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

	// Setup all Controllers
	if err := controller.AddToManager(mgr, cconfig); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create Service object to expose the metrics port.
	//servicePorts := []core.ServicePort{
	//	{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: core.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
	//}
	//_, err = metrics.CreateMetricsService(ctx, nil, servicePorts)
	//if err != nil {
	//	log.Info(err.Error())
	//}

	log.Info("Starting the Cmd.")

	// Recover panics to serialize output to json for our logger
	defer func() {
		if r := recover(); r != nil {
			log.Error(
				err, "panic",
				"recovered", r,
				"stacktrace", string(debug.Stack()),
			)
			os.Exit(2)
		}
	}()

	// Start the Cmd
	if err = mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
