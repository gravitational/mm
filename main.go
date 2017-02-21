package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/Sirupsen/logrus"
	v1 "k8s.io/client-go/1.4/pkg/api/v1"

	"github.com/gravitational/kingpin"
	"github.com/gravitational/trace"

	"github.com/gravitational/mm/pkg/constants"
	"github.com/gravitational/mm/pkg/influxdb"
	"github.com/gravitational/mm/pkg/kubernetes"
	"github.com/gravitational/mm/pkg/prometheus"
	"github.com/gravitational/mm/pkg/util"
	influx "github.com/influxdata/influxdb/client"
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

	influxService, err := op.GetService(cfg.InfluxDBServiceNamespace, cfg.InfluxDBServiceName)
	if err != nil {
		return trace.Wrap(err)
	}
	u, err := url.Parse(fmt.Sprintf("http://%s:%v", nodeAddress, influxService.Spec.Ports[0].Port))
	if err != nil {
		return trace.Wrap(err)
	}
	influxClient, err := influxdb.NewClient(influx.Config{URL: *u}, "mydb", "")
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
		if len(service.Spec.Ports) == 0 {
			return errors.New("sadffsa")
		}

		port := service.Spec.Ports[0]
		metricsURL := fmt.Sprintf("http://%s:%v/metrics", nodeAddress, port.Port)
		resp, err := util.DoHTTPRequest("GET", metricsURL, nil)
		if err != nil {
			return trace.Wrap(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return trace.Wrap(fmt.Errorf("%s returned HTTP status %s", metricsURL, resp.Status))
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading body: %s", err)
		}

		metrics, err := prometheus.Parse(body, resp.Header)
		if err != nil {
			return fmt.Errorf("error reading metrics for %s: %s", metricsURL, err)
		}

		err = influxClient.Send(metrics)
		if err != nil {
			return trace.Wrap(err)
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-signalChan
		log.Infof(fmt.Sprintf("Captured %v. Exiting...", s))
		watcher.Stop()

		switch s {
		case syscall.SIGINT:
			os.Exit(130)
		case syscall.SIGTERM:
			os.Exit(0)
		}
	}()

	return nil
}
