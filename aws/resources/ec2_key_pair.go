package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// EC2KeyPairsAPI defines the interface for EC2 Key Pairs operations.
type EC2KeyPairsAPI interface {
	DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error)
}

// NewEC2KeyPairs creates a new EC2 Key Pairs resource using the generic resource pattern.
func NewEC2KeyPairs() AwsResource {
	return NewAwsResource(&resource.Resource[EC2KeyPairsAPI]{
		ResourceTypeName: "ec2-keypairs",
		BatchSize:        200,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2KeyPairsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2KeyPairs
		},
		Lister:             listEC2KeyPairs,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2KeyPair),
		PermissionVerifier: verifyEC2KeyPairPermission,
	})
}

// listEC2KeyPairs retrieves all EC2 key pairs that match the config filters.
func listEC2KeyPairs(ctx context.Context, client EC2KeyPairsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, err
	}

	var ids []*string
	for _, keyPair := range result.KeyPairs {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: keyPair.KeyName,
			Time: keyPair.CreateTime,
			Tags: util.ConvertTypesTagsToMap(keyPair.Tags),
		}) {
			ids = append(ids, keyPair.KeyPairId)
		}
	}

	return ids, nil
}

// deleteEC2KeyPair deletes a single EC2 key pair.
func deleteEC2KeyPair(ctx context.Context, client EC2KeyPairsAPI, id *string) error {
	_, err := client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyPairId: id,
	})
	return err
}

// verifyEC2KeyPairPermission performs a dry-run delete to check permissions.
func verifyEC2KeyPairPermission(ctx context.Context, client EC2KeyPairsAPI, id *string) error {
	_, err := client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyPairId: id,
		DryRun:    aws.Bool(true),
	})
	return err
}
