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

func (op *Operator) GetNodeIP(name string) (string, error) {
	var node *v1.Node
	var err error
	if name == "" {
		nodes, err := op.ListNodes(nil)
		if err != nil {
			return "", trace.Wrap(err)
		}
		if len(nodes.Items) == 0 {
			return "", trace.Errorf("no nodes were found")
		}
		node = &nodes.Items[0]
	} else {
		node, err = op.GetNode(name)
		if err != nil {
			return "", trace.Wrap(err)
		}
	}

	var nodeIP string
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeInternalIP {
			nodeIP = address.Address
			break
		}
	}
	if nodeIP == "" {
		return "", trace.Errorf("NodeInternalIP can't be empty")
	}
	return nodeIP, nil
}

func (op *Operator) WatchServices(namespace string, labelsMap map[string]string) (watch.Interface, error) {
	watcher, err := op.Client.Core().Services(constants.Namespace(namespace)).Watch(api.ListOptions{LabelSelector: GetLabelSelector(labelsMap)})
	if err != nil {
		return nil, convertErr(err)
	}
	return watcher, nil
}

func (op *Operator) GetService(namespace string, name string) (*v1.Service, error) {
	svc, err := op.Client.Core().Services(constants.Namespace(namespace)).Get(name)
	if err != nil {
		return nil, convertErr(err)
	}
	return svc, nil
}

func (op *Operator) GetNode(name string) (*v1.Node, error) {
	n, err := op.Client.Core().Nodes().Get(name)
	if err != nil {
		return nil, convertErr(err)
	}
	return n, nil
}

func GetLabelSelector(labelsMap map[string]string) labels.Selector {
	set := make(labels.Set)
	for key, val := range labelsMap {
		set[key] = val
	}
	return set.AsSelector()
}

func ExtractServiceNodePort(svc *v1.Service, port int32) (int32, error) {
	for _, p := range svc.Spec.Ports {
		if p.NodePort > 0 && p.Port == port {
			return p.NodePort, nil
		}
	}
	return 0, trace.Errorf("missing nodeport for port %v", port)
}
