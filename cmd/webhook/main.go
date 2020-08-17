package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	slo "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	monitoring "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	"github.com/spf13/pflag"
	"go.medium.engineering/picchu/pkg/apis"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/client/scheme"
	"go.medium.engineering/picchu/pkg/webhook"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	core "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	webhookPort := pflag.Int("webhook-port", 8443, "The port to listen for admission webhooks on")

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
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

	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		panic(err)
	}

	webhook.Init(cli, int32(*webhookPort), namespace, log)

	ctx := context.Background()

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

	mgr.GetWebhookServer().Port = *webhookPort

	webhook.Register(mgr)

	// Create Service object to expose the metrics port.
	servicePorts := []core.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: core.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
	}
	_, err = metrics.CreateMetricsService(ctx, nil, servicePorts)
	if err != nil {
		log.Info(err.Error())
	}

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
