package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/Sirupsen/logrus"

	"github.com/gravitational/kingpin"
	"github.com/gravitational/trace"

	"io/ioutil"

	"github.com/gravitational/mm/pkg/constants"
	"github.com/gravitational/mm/pkg/kubernetes"
	"github.com/gravitational/mm/pkg/util"
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
	var cfg constants.CommandLineFlags
	kingpin.Flag(constants.FlagLogLevel, "Log level.").Default("info").Envar(constants.EnvLogLevel).EnumVar(&cfg.LogLevel, "debug", "info", "warning", "error", "fatal", "panic")
	kingpin.Flag(constants.FlagKubeConfig, "Path to kubeconfig.").Default(filepath.Join(os.Getenv("HOME"), ".kube", "config")).Envar(constants.EnvKubeConfig).StringVar(&cfg.KubeConfig)
	kingpin.Parse()
	return cfg
}

func run(cfg constants.CommandLineFlags) error {
	client, config, err := kubernetes.GetClient(cfg.KubeConfig)
	if err != nil {
		return trace.Wrap(err)
	}

	l, err := kubernetes.NewServiceLocator(kubernetes.ServiceLocatorConfig{
		Client: client,
		Config: config,
	})
	if err != nil {
		return trace.Wrap(err)
	}

	services, err := l.List("monitoring", map[string]string{"app": "node-exporter"})
	if err != nil {
		return trace.Wrap(err)
	}

	for _, service := range services.Items {
		data := service.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
		svc := kubernetes.Service{}
		json.Unmarshal([]byte(data), &svc)
		fmt.Println(svc)
		for _, port := range svc.Spec.Ports {
			resp, err := util.DoHTTPRequest("GET", fmt.Sprintf("http://192.168.99.100:%v/metrics", port["port"]), nil)
			if err != nil {
				return trace.Wrap(err)
			}
			out, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return trace.Wrap(err)
			}
			fmt.Println(string(out))
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
