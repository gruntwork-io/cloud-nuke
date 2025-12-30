package gcp

import (
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
)

// Type aliases to expose gcp/resources types at the gcp package level.
// This allows external packages to use gcp.GcpResource, gcp.GcpResources, etc.
// while keeping the actual definitions in gcp/resources to avoid import cycles.

type (
	// GcpResource is an interface that represents a single GCP resource.
	GcpResource = resources.GcpResource

	// GcpResources is a struct to hold multiple instances of GcpResource.
	GcpResources = resources.GcpResources

	// GcpProjectResources is a struct that represents the resources found in a single GCP project.
	GcpProjectResources = resources.GcpProjectResources
)
