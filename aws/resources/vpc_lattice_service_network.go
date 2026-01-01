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
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[VPCLatticeServiceNetworkAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for VPC Lattice client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = vpclattice.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.VPCLatticeServiceNetwork
		},
		Lister: listVPCLatticeServiceNetworks,
		Nuker:  resource.SequentialDeleter(deleteVPCLatticeServiceNetwork),
	})
}

// listVPCLatticeServiceNetworks retrieves all VPC Lattice Service Networks that match the config filters.
func listVPCLatticeServiceNetworks(ctx context.Context, client VPCLatticeServiceNetworkAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.ListServiceNetworks(ctx, nil)
	if err != nil {
		return nil, err
	}

	var ids []*string
	for _, item := range output.Items {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: item.Name,
			Time: item.CreatedAt,
		}) {
			ids = append(ids, item.Arn)
		}
	}

	return ids, nil
}

// nukeServiceAssociations deletes all service associations for a service network.
func nukeServiceAssociations(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	associations, err := client.ListServiceNetworkServiceAssociations(ctx, &vpclattice.ListServiceNetworkServiceAssociationsInput{
		ServiceNetworkIdentifier: id,
	})

	if err != nil {
		return err
	}

	for _, item := range associations.Items {
		_, err := client.DeleteServiceNetworkServiceAssociation(ctx, &vpclattice.DeleteServiceNetworkServiceAssociationInput{
			ServiceNetworkServiceAssociationIdentifier: item.Id,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// waitUntilAllServiceAssociationDeleted waits until all service associations are deleted.
func waitUntilAllServiceAssociationDeleted(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	for i := 0; i < 10; i++ {
		output, err := client.ListServiceNetworkServiceAssociations(ctx, &vpclattice.ListServiceNetworkServiceAssociationsInput{
			ServiceNetworkIdentifier: id,
		})

		if err != nil {
			return err
		}
		if len(output.Items) == 0 {
			return nil
		}
		logging.Info("Waiting for service associations to be deleted...")
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timed out waiting for service associations to be successfully deleted")
}

// deleteVPCLatticeServiceNetwork deletes a single VPC Lattice Service Network after removing associations.
func deleteVPCLatticeServiceNetwork(ctx context.Context, client VPCLatticeServiceNetworkAPI, id *string) error {
	// First delete all service associations
	if err := nukeServiceAssociations(ctx, client, id); err != nil {
		return err
	}

	// Wait for all associations to be deleted
	if err := waitUntilAllServiceAssociationDeleted(ctx, client, id); err != nil {
		return err
	}

	// Finally delete the service network
	_, err := client.DeleteServiceNetwork(ctx, &vpclattice.DeleteServiceNetworkInput{
		ServiceNetworkIdentifier: id,
	})
	return err
}
