package main

import (
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

	"github.com/gravitational/trace"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/gravitational/mm/pkg/constants"
	"github.com/gravitational/mm/pkg/influxdb"
	"github.com/gravitational/mm/pkg/kubernetes"
	"github.com/gravitational/mm/pkg/prometheus"
	"github.com/gravitational/mm/pkg/util"
	influx "github.com/influxdata/influxdb/client/v2"
	watch "k8s.io/client-go/1.4/pkg/watch"
)

func main() {
	cfg := argParse()
	if err := util.SetupLogging(cfg.LogLevel); err != nil {
		log.Fatal(err)
	}
	if err := run(cfg); err != nil {
		log.Fatal(trace.DebugReport(err))
	}
}

func argParse() constants.CommandLineFlags {
	cfg := constants.NewCommandLineFlags()

	kingpin.Flag(constants.FlagLogLevel, "Log level.").
		Default("info").
		Envar(constants.EnvLogLevel).
		EnumVar(&cfg.LogLevel, "debug", "info", "warning", "error", "fatal", "panic")
	kingpin.Flag(constants.FlagKubeConfig, "Path to kubeconfig.").
		Default(filepath.Join(os.Getenv("HOME"), ".kube", "config")).
		Envar(constants.EnvKubeConfig).
		StringVar(&cfg.KubeConfig)
	kingpin.Flag(constants.FlagMetricsServicesNamespace, "Kubernetes namespace for metrics services.").
		Default(constants.DefaultNamespace).
		Envar(constants.EnvMetricsServicesNamespace).
		StringVar(&cfg.MetricsServicesNamespace)
	kingpin.Flag(constants.FlagMetricsServicesLabelSelector, "Kubernetes label selector for metrics services.").
		PlaceHolder("KEY:VALUE").
		StringMapVar(&cfg.MetricsServicesLabelSelector)
	kingpin.Flag(constants.FlagInfluxDBServiceNamespace, "Kubernetes namespace for InfluxDB.").
		Default(constants.DefaultNamespace).
		Envar(constants.EnvInfluxDBServiceNamespace).
		StringVar(&cfg.InfluxDBServiceNamespace)
	kingpin.Flag(constants.FlagInfluxDBServiceName, "Kubernetes service name for InfluxDB.").
		Default(constants.DefaultInfluxDBServiceName).
		Envar(constants.EnvInfluxDBServiceName).
		StringVar(&cfg.InfluxDBServiceName)
	kingpin.Flag(constants.FlagInfluxDBDatabaseName, "InfluxDB database name.").
		Envar(constants.EnvInfluxDBDatabaseName).
		StringVar(&cfg.InfluxDBDatabaseName)

	kingpin.Parse()
	return cfg
}

func run(cfg constants.CommandLineFlags) error {
	log.Infof("Starting with config %+v", cfg)

	client, config, err := kubernetes.GetClient(cfg.KubeConfig)
	if err != nil {
		return trace.Wrap(err, "can't create kubernetes client")
	}

	op, err := kubernetes.NewOperator(kubernetes.OperatorConfig{Client: client, Config: config})
	if err != nil {
		return trace.Wrap(err, "can't create kubernetes operator instance")
	}

	nodeIP, err := op.GetNodeIP()
	if err != nil {
		return trace.Wrap(err, "can't get node IP address")
	}

	influxDBService, err := op.GetService(cfg.InfluxDBServiceNamespace, cfg.InfluxDBServiceName)
	if err != nil {
		return trace.Wrap(err, "can't find InfluxDB service")
	}

	influxDBAPIPort, err := kubernetes.ExtractServiceNodePort(influxDBService, constants.DefaultInfluxDBAPIPort)
	if err != nil {
		return trace.Wrap(err, "can't find InfluxDB HTTP API port")
	}

	u, err := url.Parse(fmt.Sprintf("http://%s:%v", nodeIP, influxDBAPIPort))
	if err != nil {
		return trace.Wrap(err)
	}

	influxClient, err := influxdb.NewClient(influx.HTTPConfig{Addr: u.String()}, cfg.InfluxDBDatabaseName, "")
	if err != nil {
		return trace.Wrap(err, "can't create InfluxDB client")
	}

	watcher, err := op.WatchServices(cfg.MetricsServicesNamespace, cfg.MetricsServicesLabelSelector)
	if err != nil {
		return trace.Wrap(err, "can't watch for services labeles as %v", cfg.MetricsServicesLabelSelector)
	}
	for event := range watcher.ResultChan() {
		log.Debugf("Event: %s", event.Type)
		if event.Type != watch.Added && event.Type != watch.Modified {
			continue
		}

		service := event.Object.(*v1.Service)
		metricsURL := fmt.Sprintf("http://%s:%v/metrics", nodeIP, service.Spec.Ports[0].Port)
		log.Debugf("Fetch metrics: %s", metricsURL)

		resp, err := util.DoHTTPRequest("GET", metricsURL, nil)
		if err != nil {
			return trace.Wrap(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return trace.Errorf("%s returned HTTP status %s", metricsURL, resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return trace.Wrap(err, "error reading body")
		}

		metrics, err := prometheus.Parse(body, resp.Header)
		if err != nil {
			return trace.Wrap(err, "error reading metrics for %s", metricsURL)
		}

		err = influxClient.Send(metrics)
		if err != nil {
			return trace.Wrap(err, "error sending metrics")
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
