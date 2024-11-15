package resources

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SnapshotAPI interface {
	DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error)
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
	DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error)
}

// Snapshots - represents all user owned Snapshots
type Snapshots struct {
	BaseAwsResource
	Client      SnapshotAPI
	Region      string
	SnapshotIds []string
}

func (s *Snapshots) InitV2(cfg aws.Config) {
	s.Client = ec2.NewFromConfig(cfg)
}

func (s *Snapshots) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (s *Snapshots) ResourceName() string {
	return "snap"
}

// ResourceIdentifiers - The Snapshot ids
func (s *Snapshots) ResourceIdentifiers() []string {
	return s.SnapshotIds
}

func (s *Snapshots) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (s *Snapshots) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Snapshots
}

func (s *Snapshots) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := s.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	s.SnapshotIds = aws.ToStringSlice(identifiers)
	return s.SnapshotIds, nil
}

// Nuke - nuke 'em all!!!
func (s *Snapshots) Nuke(identifiers []string) error {
	if err := s.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
