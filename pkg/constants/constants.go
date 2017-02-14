package constants

const (
	ChangesetGroup   = "changeset.gravitational.io"
	ChangesetVersion = "v1"
	DefaultNamespace = "default"
)

// Namespace returns a default namespace if the specified namespace is empty
func Namespace(namespace string) string {
	if namespace == "" {
		return DefaultNamespace
	}
	return namespace
}
