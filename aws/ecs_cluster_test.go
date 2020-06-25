package aws

import (
	"testing"
)

// Test that we can succesfully list ECS clusters by manually creating a cluster and then using the list function to find it.
func TestListEcsClusters(t *testing.T) {
	// create ECS cluster in a specific region
	// call api to list all clusters
	// assert that result contains your new cluster
}

// Test that we can filter ECS clusters by 'created_at' tag value.
func TestFilterEcsClusters(t *testing.T) {
	// create ECS cluster in a specific region with a tag on creation (very short time frame)
	// call api to list all clusters
	// filter results - possibly using api
	// assert that result contains your new cluster only (provided it's the only one created in the short time frame)
}

// Test that we can delete ECS clusters by manually creating an ECS cluster, and then deleting it using the nuke function.
func TestNukeEcsClusters(t *testing.T) {
	// create ECS cluster
	// list all ECS clusters
	// nuke all ECS clusters
	// list all ECS clusters
	// assert no ECS clusters left
}

// Test that we can delete tagged ECS clusters by manually creating an ECS cluster, and then deleting it using the nuke function.
func TestNukeEcsClustersByTag(t *testing.T) {
	// create ECS cluster with a 'created_at' tag
	// list all ECS clusters with this tag & filter
	// nuke ECS cluster by tag and filter
	// list all ECS clusters
	// assert no ECS clusters as filter criteria
}
