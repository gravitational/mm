package constants

const (
	EnvLogLevel   = "MM_LOG_LEVEL"
	EnvKubeConfig = "MM_KUBE_CONFIG"
	EnvNamespace  = "MM_NAMESPACE"
)

const (
	FlagLogLevel      = "log-level"
	FlagKubeConfig    = "kubeconfig"
	FlagNamespace     = "namespace"
	FlagLabelSelector = "label-selector"
)

type CommandLineFlags struct {
	LogLevel      string
	KubeConfig    string
	Namespace     string
	LabelSelector map[string]string
}

func NewCommandLineFlags() *CommandLineFlags {
	return &CommandLineFlags{
		LabelSelector: make(map[string]string),
	}
}
