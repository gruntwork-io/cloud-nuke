package gcp

// An abstract GCP Resource
type GcpResource interface {
	// Friendly name for what kind of resource this is
	Kind() string
	// The name that identifies this resources for the location
	Name() string
	// The zone this resource is located in. Empty if the kind of resource kind
	// is not located by zone.
	Zone() string
	// The region this resource is located in
	Region() string
	// Destroys the resource
	Nuke(ctx *GcpContext) error
}
