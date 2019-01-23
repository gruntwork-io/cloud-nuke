package gcp

import (
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	google "golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"net/http"
	"time"
)

type GcpContext struct {
	Client  *http.Client
	Service *compute.Service
	Project string
	Regions []*compute.Region
}

func DefaultContext() (*GcpContext, error) {
	creds, err := google.FindDefaultCredentials(oauth2.NoContext, compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	client := oauth2.NewClient(context.Background(), creds.TokenSource)

	service, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	regions, err := service.Regions.List(creds.ProjectID).Do()
	if err != nil {
		return nil, err
	}

	context := &GcpContext{
		Client:  client,
		Service: service,
		Project: creds.ProjectID,
		Regions: regions.Items,
	}

	return context, nil
}

func (ctx *GcpContext) ContainsRegion(region string) bool {
	for _, r := range ctx.Regions {
		if r.Name == region {
			return true
		}
	}

	return false
}

func (ctx *GcpContext) GetAllResources(excludedRegions []string, excludeAfter time.Time) ([]GcpResource, error) {
	resources := []GcpResource{}

	instances, err := GetAllGceInstances(ctx, excludedRegions, excludeAfter)
	if err != nil {
		return nil, err
	}

	resources = append(resources, instances...)

	return resources, nil
}

type NukeWorkerResult struct {
	Resource GcpResource
	Err      error
}

func nukeWorker(ctx *GcpContext, resource GcpResource, output chan<- NukeWorkerResult) {
	err := resource.Nuke(ctx)
	result := NukeWorkerResult{
		Resource: resource,
		Err:      err,
	}
	output <- result
}

func (ctx *GcpContext) NukeAllResources(resources []GcpResource) []error {
	nukeErrors := []error{}
	batchSize := 5
	results := make(chan NukeWorkerResult, 100)

	if len(resources) < batchSize {
		batchSize = len(resources)
	}

	for i := 0; i < batchSize; i++ {
		go nukeWorker(ctx, resources[i], results)
	}

	for i := 0; i < len(resources); i++ {
		result := <-results
		if result.Err != nil {
			logging.Logger.Warnf("Could not delete resource: %s: %s Region=%s Zone=%s \n%s",
				result.Resource.Kind(), result.Resource.Name(), result.Resource.Region(),
				result.Resource.Zone(),
				errors.WithStackTrace(result.Err).Error())
			nukeErrors = append(nukeErrors, result.Err)
		}

		next := i + batchSize
		if next < len(resources) {
			go nukeWorker(ctx, resources[next], results)
		}
	}

	return nukeErrors
}
