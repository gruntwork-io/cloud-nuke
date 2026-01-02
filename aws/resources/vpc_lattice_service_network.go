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
		// Service network deletion requires: delete associations → wait → delete network
		Nuker: resource.MultiStepDeleter(
			deleteServiceAssociations,
			waitForServiceAssociationsDeleted,
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
			if cfg.ShouldInclude(config.ResourceValue{
				Name: item.Name,
				Time: item.CreatedAt,
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

// waitForServiceAssociationsDeleted polls until all service associations are deleted.
// Times out after 100 seconds.
func waitForServiceAssociationsDeleted(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	for i := 0; i < 10; i++ {
		output, err := client.ListServiceNetworkServiceAssociations(ctx, &vpclattice.ListServiceNetworkServiceAssociationsInput{
			ServiceNetworkIdentifier: id,
		})
		if err != nil {
			// Error likely means service network doesn't exist or associations are gone
			return nil //nolint:nilerr
		}
		if len(output.Items) == 0 {
			return nil
		}
		logging.Debugf("Waiting for service associations to be deleted for service network %s...", aws.ToString(id))
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timed out waiting for service associations to be deleted for service network %s", aws.ToString(id))
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
