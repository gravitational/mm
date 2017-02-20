package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/Sirupsen/logrus"
	v1 "k8s.io/client-go/1.4/pkg/api/v1"

	"github.com/gravitational/kingpin"
	"github.com/gravitational/trace"

	"github.com/gravitational/mm/pkg/constants"
	"github.com/gravitational/mm/pkg/kubernetes"
	"github.com/gravitational/mm/pkg/util"
	"github.com/prometheus/common/expfmt"
	watch "k8s.io/client-go/1.4/pkg/watch"
)

func main() {
	cfg := argParse()
	if err := util.SetupLogging(cfg.LogLevel); err != nil {
		log.Fatal(trace.DebugReport(err))
	}
	if err := run(cfg); err != nil {
		log.Fatal(trace.DebugReport(err))
	}
}

func argParse() constants.CommandLineFlags {
	cfg := constants.NewCommandLineFlags()
	kingpin.Flag(constants.FlagLogLevel, "Log level.").Default("info").Envar(constants.EnvLogLevel).EnumVar(&cfg.LogLevel, "debug", "info", "warning", "error", "fatal", "panic")
	kingpin.Flag(constants.FlagKubeConfig, "Path to kubeconfig.").Default(filepath.Join(os.Getenv("HOME"), ".kube", "config")).Envar(constants.EnvKubeConfig).StringVar(&cfg.KubeConfig)
	kingpin.Flag(constants.FlagMetricsServicesNamespace, "Kubernetes namespace for metrics services.").Default(constants.DefaultNamespace).Envar(constants.EnvMetricsServicesNamespace).StringVar(&cfg.MetricsServicesNamespace)
	kingpin.Flag(constants.FlagMetricsServicesLabelSelector, "Kubernetes label selector for metrics services.").PlaceHolder("KEY:VALUE").StringMapVar(&cfg.MetricsServicesLabelSelector)
	kingpin.Flag(constants.FlagInfluxDBServiceNamespace, "Kubernetes namespace for InfluxDB.").Default(constants.DefaultNamespace).Envar(constants.EnvInfluxDBServiceNamespace).StringVar(&cfg.InfluxDBServiceNamespace)
	kingpin.Flag(constants.FlagInfluxDBServiceName, "Kubernetes service name for InfluxDB.").Default(constants.DefaultInfluxDBServiceName).Envar(constants.EnvInfluxDBServiceName).StringVar(&cfg.InfluxDBServiceName)
	kingpin.Parse()
	return cfg
}

func run(cfg constants.CommandLineFlags) error {
	log.Infof("Starting with config %+v", cfg)

	client, config, err := kubernetes.GetClient(cfg.KubeConfig)
	if err != nil {
		return trace.Wrap(err)
	}

	op, err := kubernetes.NewOperator(kubernetes.OperatorConfig{Client: client, Config: config})
	if err != nil {
		return trace.Wrap(err)
	}

	nodeAddress, err := op.GetNodeInternalIP()
	if err != nil {
		return trace.Wrap(err)
	}

	watcher, err := op.WatchServices(cfg.MetricsServicesNamespace, cfg.MetricsServicesLabelSelector)
	if err != nil {
		return trace.Wrap(err)
	}
	for event := range watcher.ResultChan() {
		if event.Type != watch.Added && event.Type != watch.Modified {
			continue
		}
		service := event.Object.(*v1.Service)
		for _, port := range service.Spec.Ports {
			resp, err := util.DoHTTPRequest("GET", fmt.Sprintf("http://%s:%v/metrics", nodeAddress, port.Port), nil)
			if err != nil {
				return trace.Wrap(err)
			}
			parser := &expfmt.TextParser{}
			metrics, err := parser.TextToMetricFamilies(resp.Body)
			if err != nil {
				return trace.Wrap(err)
			}
			for _, mf := range metrics {
				fmt.Println(mf.GetMetric())
			}
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-signalChan
		log.Println(fmt.Sprintf("Captured %v. Exiting...", s))

		switch s {
		case syscall.SIGINT:
			os.Exit(130)
		case syscall.SIGTERM:
			os.Exit(0)
		}
	}()

	return nil
}
