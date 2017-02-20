package kubernetes

import (
	"errors"

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

type OperatorConfig struct {
	// Client is k8s client
	Client *kubernetes.Clientset
	// Config is rest client config
	Config *rest.Config
}

func (c *OperatorConfig) CheckAndSetDefaults() error {
	if c.Client == nil {
		return trace.BadParameter("missing parameter Client")
	}
	if c.Config == nil {
		return trace.BadParameter("missing parameter Config")
	}
	return nil
}

type Operator struct {
	OperatorConfig
	client *rest.RESTClient
}

func NewOperator(config OperatorConfig) (*Operator, error) {
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

	return &Operator{OperatorConfig: config, client: clt}, nil
}

func (op *Operator) ListNodes(labelsMap map[string]string) (*v1.NodeList, error) {
	nodes, err := op.Client.Core().Nodes().List(api.ListOptions{LabelSelector: GetLabelSelector(labelsMap)})
	if err != nil {
		return nil, convertErr(err)
	}

	return nodes, nil
}

func (op *Operator) GetNodeInternalIP() (string, error) {
	nodes, err := op.ListNodes(nil)
	if err != nil {
		return "", trace.Wrap(err)
	}
	if len(nodes.Items) == 0 {
		return "", errors.New("no nodes were found")
	}

	var nodeAddress string
	for _, address := range nodes.Items[0].Status.Addresses {
		if address.Type == v1.NodeInternalIP {
			nodeAddress = address.Address
		}
	}
	if nodeAddress == "" {
		return "", errors.New("can't find NodeInternalIP")
	}
	return nodeAddress, nil
}

func (l *Operator) WatchServices(namespace string, labelsMap map[string]string) (watch.Interface, error) {
	watcher, err := l.Client.Core().Services(constants.Namespace(namespace)).Watch(api.ListOptions{LabelSelector: GetLabelSelector(labelsMap)})
	if err != nil {
		return nil, convertErr(err)
	}

	return watcher, nil
}

func GetLabelSelector(labelsMap map[string]string) labels.Selector {
	set := make(labels.Set)
	for key, val := range labelsMap {
		set[key] = val
	}

	return set.AsSelector()
}
