package gcp

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/util"
	gruntworkerrors "github.com/gruntwork-io/gruntwork-cli/errors"
	terratestGcp "github.com/gruntwork-io/terratest/modules/gcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	compute "google.golang.org/api/compute/v1"
	"strings"
	"testing"
	"time"
)

// Whether this list of resources contains a resource with the given name
func resourcesContains(resources []GcpResource, zone string, name string) bool {
	for _, resource := range resources {
		if resource.Name() == name && resource.Zone() == zone {
			return true
		}
	}

	return false
}

// Create a compute instance with the given name and protection status
func createTestInstance(ctx *GcpContext, name string, zone string, protected bool) error {
	machineType := fmt.Sprintf("projects/%s/zones/%s/machineTypes/f1-micro", ctx.Project, zone)

	iface := &compute.NetworkInterface{}

	initializeParams := &compute.AttachedDiskInitializeParams{
		SourceImage: "projects/debian-cloud/global/images/debian-9-stretch-v20181113",
	}

	disk := &compute.AttachedDisk{
		AutoDelete:       true,
		Boot:             true,
		InitializeParams: initializeParams,
	}

	instance := &compute.Instance{
		Name:               name,
		MachineType:        machineType,
		DeletionProtection: protected,
		NetworkInterfaces:  []*compute.NetworkInterface{iface},
		Disks:              []*compute.AttachedDisk{disk},
	}

	_, err := ctx.Service.Instances.Insert(ctx.Project, zone, instance).Do()
	return err
}

func cleanupInstances(t *testing.T, ctx *GcpContext, zone string, names []string) {
	for _, name := range names {
		call := ctx.Service.Instances.SetDeletionProtection(ctx.Project, zone, name)
		call.DeletionProtection(false)

		_, err := call.Do()
		if err != nil {
			t.Logf("Warning: could not unset deletion protection on instance: %s %s", name, gruntworkerrors.WithStackTrace(err).Error())
		}

		_, err = ctx.Service.Instances.Delete(ctx.Project, zone, name).Do()
		if err != nil {
			t.Logf("Warning: could not delete instance: %s %s", name, gruntworkerrors.WithStackTrace(err).Error())
		}
	}
}

// Test that the context correctly chaches regions and can get zone names from
// zone urls
func TestRegionZones(t *testing.T) {
	t.Parallel()

	ctx, err := DefaultContext()
	require.NoError(t, err)

	for _, region := range ctx.Regions {
		for _, zoneUrl := range region.Zones {
			zone, err := zoneFromUrl(zoneUrl)
			if !assert.NoError(t, err) {
				continue
			}

			regionName, err := regionFromZone(ctx, zone)
			if !assert.NoError(t, err) {
				continue
			}

			assert.Equal(t, region.Name, regionName)
		}
	}
}

// Create several instances and test that they can be nuked
func TestNukeInstances(t *testing.T) {
	t.Parallel()

	ctx, err := DefaultContext()
	require.NoError(t, err)

	zone := terratestGcp.GetRandomZone(t, ctx.Project, nil, nil, nil)

	instanceName := strings.ToLower("cloud-nuke-test-" + util.UniqueID())
	protectedInstanceName := strings.ToLower("cloud-nuke-test-" + util.UniqueID())

	defer cleanupInstances(t, ctx, zone, []string{instanceName, protectedInstanceName})

	err = createTestInstance(ctx, instanceName, zone, false)
	require.NoError(t, err)

	err = createTestInstance(ctx, protectedInstanceName, zone, true)
	require.NoError(t, err)

	instances, err := GetAllGceInstances(ctx, []string{}, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.True(t, resourcesContains(instances, zone, instanceName),
		"the created instance should show up in the list of instances")
	assert.False(t, resourcesContains(instances, zone, protectedInstanceName),
		"the protected instance should not show up in the list of instances")

	nukeErrors := ctx.NukeAllResources(instances)
	if len(nukeErrors) != 0 {
		for _, err = range nukeErrors {
			t.Logf(gruntworkerrors.WithStackTrace(err).Error())
		}
		assert.FailNow(t, "Some resources failed to nuke.")
	}

	// status doesn't update immediately, so give it a minute or two to show it
	// is terminating
	lastStatus := ""
	for tries := 0; tries < 40; tries++ {
		instance, err := ctx.Service.Instances.Get(ctx.Project, zone, instanceName).Do()
		require.NoError(t, err)

		lastStatus = instance.Status

		if instance.Status == "TERMINATED" {
			break
		}

		time.Sleep(3 * time.Second)
	}

	require.Equal(t, "TERMINATED", lastStatus,
		"instance should terminate after it is nuked within two minutes")
}
