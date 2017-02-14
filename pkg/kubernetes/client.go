package kubernetes

import (
	"fmt"
	"net/http"

	"github.com/gravitational/trace"

	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api/errors"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/tools/clientcmd"
)

func GetClient(configPath string) (*kubernetes.Clientset, *rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, nil, trace.Wrap(err)
		}
		return client, config, nil
	}

	config, err = clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}
	return client, config, nil
}

func convertErr(err error) error {
	if err == nil {
		return nil
	}
	se, ok := err.(*errors.StatusError)
	if !ok {
		return err
	}

	format, args := se.DebugError()
	status := se.Status()
	switch {
	case status.Code == http.StatusConflict && status.Reason == unversioned.StatusReasonAlreadyExists:
		return trace.AlreadyExists("error: %v, details: %v", err.Error(), fmt.Sprintf(format, args...))
	case status.Code == http.StatusNotFound:
		return trace.NotFound("error: %v, details: %v", err.Error(), fmt.Sprintf(format, args...))
	}
	return err
}
