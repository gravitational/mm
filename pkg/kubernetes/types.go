package kubernetes

type Metadata struct {
	Name   string
	Labels map[string]string
}

type Port map[string]interface{}

type Spec struct {
	Ports []Port
}

type Service struct {
	Metadata Metadata
	Spec     Spec
}
