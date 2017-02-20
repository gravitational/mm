package constants

const DefaultInfluxDBServiceName = "influxdb"

const (
	EnvLogLevel                 = "MM_LOG_LEVEL"
	EnvKubeConfig               = "MM_KUBE_CONFIG"
	EnvMetricsServicesNamespace = "MM_METRICS_SERVICES_NAMESPACE"
	EnvInfluxDBServiceNamespace = "MM_INFLUXDB_SERVICE_NAMESPACE"
	EnvInfluxDBServiceName      = "MM_INFLUXDB_SERVICE_NAME"
)

const (
	FlagLogLevel                     = "log-level"
	FlagKubeConfig                   = "kubeconfig"
	FlagMetricsServicesNamespace     = "metrics-services-namespace"
	FlagMetricsServicesLabelSelector = "metrics-services-label-selector"
	FlagInfluxDBServiceNamespace     = "influxdb-service-namespace"
	FlagInfluxDBServiceName          = "influxdb-service-name"
)

type CommandLineFlags struct {
	LogLevel                     string
	KubeConfig                   string
	MetricsServicesNamespace     string
	MetricsServicesLabelSelector map[string]string
	InfluxDBServiceNamespace     string
	InfluxDBServiceName          string
}

func NewCommandLineFlags() CommandLineFlags {
	return CommandLineFlags{
		MetricsServicesLabelSelector: make(map[string]string),
	}
}
