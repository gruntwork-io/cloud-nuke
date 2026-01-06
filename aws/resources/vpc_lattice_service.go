package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// VPCLatticeServiceAPI defines the interface for VPC Lattice Service operations.
type VPCLatticeServiceAPI interface {
	ListServices(ctx context.Context, params *vpclattice.ListServicesInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServicesOutput, error)
	DeleteService(ctx context.Context, params *vpclattice.DeleteServiceInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceOutput, error)
	ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error)
	DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error)
}

// NewVPCLatticeService creates a new VPC Lattice Service resource using the generic resource pattern.
func NewVPCLatticeService() AwsResource {
	return NewAwsResource(&resource.Resource[VPCLatticeServiceAPI]{
		ResourceTypeName: "vpc-lattice-service",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[VPCLatticeServiceAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = vpclattice.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.VPCLatticeService
		},
		Lister: listVPCLatticeServices,
		// VPC Lattice Service deletion requires: delete associations → wait → delete service
		Nuker: resource.MultiStepDeleter(
			deleteVPCLatticeServiceAssociations,
			waitForVPCLatticeServiceAssociationsDeleted,
			deleteVPCLatticeService,
		),
	})
}

// listVPCLatticeServices retrieves all VPC Lattice Services that match the config filters.
func listVPCLatticeServices(ctx context.Context, client VPCLatticeServiceAPI, _ resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allServices []*string

	paginator := vpclattice.NewListServicesPaginator(client, &vpclattice.ListServicesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, service := range page.Items {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: service.Name,
				Time: service.CreatedAt,
			}) {
				allServices = append(allServices, service.Arn)
			}
		}
	}

	return allServices, nil
}

// deleteVPCLatticeServiceAssociations deletes all service network associations for the given service.
// VPC Lattice Service cannot be deleted while associations exist.
func deleteVPCLatticeServiceAssociations(ctx context.Context, client VPCLatticeServiceAPI, serviceID *string) error {
	paginator := vpclattice.NewListServiceNetworkServiceAssociationsPaginator(client, &vpclattice.ListServiceNetworkServiceAssociationsInput{
		ServiceIdentifier: serviceID,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, assoc := range page.Items {
			logging.Debugf("Deleting service network association %s for VPC Lattice Service %s",
				aws.ToString(assoc.Id), aws.ToString(serviceID))
			if _, err := client.DeleteServiceNetworkServiceAssociation(ctx, &vpclattice.DeleteServiceNetworkServiceAssociationInput{
				ServiceNetworkServiceAssociationIdentifier: assoc.Id,
			}); err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}

// waitForVPCLatticeServiceAssociationsDeleted polls until all associations for the given service are deleted.
// Times out after 100 seconds.
func waitForVPCLatticeServiceAssociationsDeleted(ctx context.Context, client VPCLatticeServiceAPI, serviceID *string) error {
	for i := 0; i < 10; i++ {
		output, err := client.ListServiceNetworkServiceAssociations(ctx, &vpclattice.ListServiceNetworkServiceAssociationsInput{
			ServiceIdentifier: serviceID,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
		if len(output.Items) == 0 {
			return nil
		}
		logging.Debugf("Waiting for service associations to be deleted for VPC Lattice Service %s...", aws.ToString(serviceID))
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timed out waiting for service associations to be deleted for VPC Lattice Service %s", aws.ToString(serviceID))
}

// deleteVPCLatticeService deletes the VPC Lattice Service.
func deleteVPCLatticeService(ctx context.Context, client VPCLatticeServiceAPI, serviceID *string) error {
	_, err := client.DeleteService(ctx, &vpclattice.DeleteServiceInput{
		ServiceIdentifier: serviceID,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
