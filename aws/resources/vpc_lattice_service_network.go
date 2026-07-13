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

// VPCLatticeServiceNetworkAPI defines the interface for VPC Lattice Service Network operations.
type VPCLatticeServiceNetworkAPI interface {
	ListServiceNetworks(ctx context.Context, params *vpclattice.ListServiceNetworksInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworksOutput, error)
	DeleteServiceNetwork(ctx context.Context, params *vpclattice.DeleteServiceNetworkInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkOutput, error)
	ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error)
	DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error)
	ListServiceNetworkVpcAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkVpcAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkVpcAssociationsOutput, error)
	DeleteServiceNetworkVpcAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkVpcAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkVpcAssociationOutput, error)
	ListTagsForResource(ctx context.Context, params *vpclattice.ListTagsForResourceInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTagsForResourceOutput, error)
}

// NewVPCLatticeServiceNetwork creates a new VPC Lattice Service Network resource using the generic resource pattern.
func NewVPCLatticeServiceNetwork() AwsResource {
	return NewAwsResource(&resource.Resource[VPCLatticeServiceNetworkAPI]{
		ResourceTypeName: "vpc-lattice-service-network",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[VPCLatticeServiceNetworkAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = vpclattice.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.VPCLatticeServiceNetwork
		},
		Lister: listVPCLatticeServiceNetworks,
		// A service network cannot be deleted while it still has service or VPC
		// associations, so delete both, wait for them to clear, then delete it.
		Nuker: resource.MultiStepDeleter(
			deleteServiceAssociations,
			deleteVpcAssociations,
			waitForAssociationsDeleted,
			deleteServiceNetwork,
		),
	})
}

// listVPCLatticeServiceNetworks retrieves all VPC Lattice Service Networks that match the config filters.
func listVPCLatticeServiceNetworks(ctx context.Context, client VPCLatticeServiceNetworkAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string

	paginator := vpclattice.NewListServiceNetworksPaginator(client, &vpclattice.ListServiceNetworksInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, item := range page.Items {
			tagsOutput, err := client.ListTagsForResource(ctx, &vpclattice.ListTagsForResourceInput{
				ResourceArn: item.Arn,
			})
			if err != nil {
				logging.Debugf("Failed to get tags for VPC Lattice Service Network %s: %s", aws.ToString(item.Arn), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: item.Name,
				Time: item.CreatedAt,
				Tags: tagsOutput.Tags,
			}) {
				ids = append(ids, item.Arn)
			}
		}
	}

	return ids, nil
}

// deleteServiceAssociations deletes all service associations for the given service network.
// Service network cannot be deleted while associations exist.
func deleteServiceAssociations(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	paginator := vpclattice.NewListServiceNetworkServiceAssociationsPaginator(client, &vpclattice.ListServiceNetworkServiceAssociationsInput{
		ServiceNetworkIdentifier: id,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, item := range page.Items {
			logging.Debugf("Deleting service association %s for service network %s", aws.ToString(item.Id), aws.ToString(id))
			if _, err := client.DeleteServiceNetworkServiceAssociation(ctx, &vpclattice.DeleteServiceNetworkServiceAssociationInput{
				ServiceNetworkServiceAssociationIdentifier: item.Id,
			}); err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}

// deleteVpcAssociations deletes all VPC associations for the given service network.
// Service network cannot be deleted while associations exist.
func deleteVpcAssociations(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	paginator := vpclattice.NewListServiceNetworkVpcAssociationsPaginator(client, &vpclattice.ListServiceNetworkVpcAssociationsInput{
		ServiceNetworkIdentifier: id,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, item := range page.Items {
			logging.Debugf("Deleting VPC association %s for service network %s", aws.ToString(item.Id), aws.ToString(id))
			if _, err := client.DeleteServiceNetworkVpcAssociation(ctx, &vpclattice.DeleteServiceNetworkVpcAssociationInput{
				ServiceNetworkVpcAssociationIdentifier: item.Id,
			}); err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}

// waitForAssociationsDeleted polls until all service and VPC associations are
// deleted. Times out after 100 seconds.
func waitForAssociationsDeleted(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	for i := 0; i < 10; i++ {
		remaining, err := countAssociations(ctx, client, id)
		if err != nil {
			// Error likely means the service network or its associations are gone
			return nil //nolint:nilerr
		}
		if remaining == 0 {
			return nil
		}
		logging.Debugf("Waiting for associations to be deleted for service network %s...", aws.ToString(id))
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timed out waiting for associations to be deleted for service network %s", aws.ToString(id))
}

// countAssociations returns the number of service plus VPC associations still
// attached to the service network.
func countAssociations(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) (int, error) {
	services, err := client.ListServiceNetworkServiceAssociations(ctx, &vpclattice.ListServiceNetworkServiceAssociationsInput{
		ServiceNetworkIdentifier: id,
	})
	if err != nil {
		return 0, err
	}

	vpcs, err := client.ListServiceNetworkVpcAssociations(ctx, &vpclattice.ListServiceNetworkVpcAssociationsInput{
		ServiceNetworkIdentifier: id,
	})
	if err != nil {
		return 0, err
	}

	return len(services.Items) + len(vpcs.Items), nil
}

// deleteServiceNetwork deletes the VPC Lattice Service Network.
func deleteServiceNetwork(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	_, err := client.DeleteServiceNetwork(ctx, &vpclattice.DeleteServiceNetworkInput{
		ServiceNetworkIdentifier: id,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
