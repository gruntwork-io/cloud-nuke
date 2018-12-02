package gcp

type GcpResource interface {
	ResourceName() string
	Name() string
	LocationName() string
	Location() string
	Region() string
	Nuke(ctx *GcpContext) error
}
