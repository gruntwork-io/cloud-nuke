package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EBSVolumesAPI interface {
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error)
}

// EBSVolumes - represents all ebs volumes
type EBSVolumes struct {
	BaseAwsResource
	Client    EBSVolumesAPI
	Region    string
	VolumeIds []string
}

func (ev *EBSVolumes) InitV2(cfg aws.Config) {
	ev.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ev *EBSVolumes) ResourceName() string {
	return "ebs"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ev *EBSVolumes) ResourceIdentifiers() []string {
	return ev.VolumeIds
}

func (ev *EBSVolumes) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ev *EBSVolumes) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EBSVolume
}

func (ev *EBSVolumes) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ev.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ev.VolumeIds = aws.ToStringSlice(identifiers)
	return ev.VolumeIds, nil
}

// Nuke - nuke 'em all!!!
func (ev *EBSVolumes) Nuke(identifiers []string) error {
	if err := ev.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
