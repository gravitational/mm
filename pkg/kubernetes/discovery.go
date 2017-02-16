package kubernetes

import (
	"github.com/gravitational/trace"

	"github.com/gravitational/mm/pkg/constants"
	"k8s.io/client-go/1.4/kubernetes"
	api "k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	v1 "k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/labels"
	serializer "k8s.io/client-go/1.4/pkg/runtime/serializer"
	watch "k8s.io/client-go/1.4/pkg/watch"
	"k8s.io/client-go/1.4/rest"
)

type ServiceLocatorConfig struct {
	// Client is k8s client
	Client *kubernetes.Clientset
	// Config is rest client config
	Config *rest.Config
}

func (c *ServiceLocatorConfig) CheckAndSetDefaults() error {
	if c.Client == nil {
		return trace.BadParameter("missing parameter Client")
	}
	if c.Config == nil {
		return trace.BadParameter("missing parameter Config")
	}
	return nil
}

type ServiceLocator struct {
	ServiceLocatorConfig
	client *rest.RESTClient
}

func NewServiceLocator(config ServiceLocatorConfig) (*ServiceLocator, error) {
	if err := config.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	cfg := *config.Config
	cfg.APIPath = "/apis"
	if cfg.UserAgent == "" {
		cfg.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	cfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	cfg.GroupVersion = &unversioned.GroupVersion{Group: constants.ChangesetGroup, Version: constants.ChangesetVersion}
	clt, err := rest.RESTClientFor(&cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &ServiceLocator{ServiceLocatorConfig: config, client: clt}, nil
}

func (l *ServiceLocator) Watch(namespace string, labelsSet map[string]string) (watch.Interface, error) {
	set := make(labels.Set)
	for key, val := range labelsSet {
		set[key] = val
	}

	watcher, err := l.Client.Core().Services(constants.Namespace(namespace)).Watch(api.ListOptions{LabelSelector: set.AsSelector()})
	if err != nil {
		return nil, convertErr(err)
	}

	return watcher, nil
}

func WatchForService(kubeconfig, namespace string, labelSelector map[string]string) (<-chan *v1.Service, error) {
	client, config, err := GetClient(kubeconfig)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	l, err := NewServiceLocator(ServiceLocatorConfig{Client: client, Config: config})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	watcher, err := l.Watch(namespace, labelSelector)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	serviceCh := make(chan *v1.Service)
	go func() {
		for event := range watcher.ResultChan() {
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			serviceCh <- event.Object.(*v1.Service)
		}
	}()

	return serviceCh, nil
}
