package constants

const (
	EnvLogLevel   = "MM_LOG_LEVEL"
	EnvKubeConfig = "MM_KUBE_CONFIG"
)

const (
	FlagLogLevel   = "log-level"
	FlagKubeConfig = "kubeconfig"
)

type CommandLineFlags struct {
	LogLevel   string
	KubeConfig string
}
