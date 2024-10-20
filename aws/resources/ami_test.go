package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

type mockedAMI struct {
	AMIsAPI
	DeregisterImageOutput ec2.DeregisterImageOutput
	DescribeImagesOutput  ec2.DescribeImagesOutput
}

func (m mockedAMI) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	return &m.DeregisterImageOutput, nil
}

func (m mockedAMI) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOutput, nil
}

func TestAMIGetAll_SkipAWSManaged(t *testing.T) {
	t.Parallel()

	testName := "test-ami"
	testImageId1 := "test-image-id1"
	testImageId2 := "test-image-id2"
	now := time.Now()
	acm := AMIs{
		Client: mockedAMI{
			DescribeImagesOutput: ec2.DescribeImagesOutput{
				Images: []types.Image{
					{
						ImageId:      &testImageId1,
						Name:         &testName,
						CreationDate: aws.String(now.Format("2006-01-02T15:04:05.000Z")),
						Tags: []types.Tag{
							{
								Key:   aws.String("aws-managed"),
								Value: aws.String("true"),
							},
						},
					},
					{
						ImageId:      &testImageId2,
						Name:         aws.String("AwsBackup_Test"),
						CreationDate: aws.String(now.Format("2006-01-02T15:04:05.000Z")),
						Tags: []types.Tag{
							{
								Key:   aws.String("aws-managed"),
								Value: aws.String("true"),
							},
						},
					},
				},
			},
		},
	}

	amis, err := acm.getAll(context.Background(), config.Config{})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(amis), testImageId1)
	assert.NotContains(t, aws.ToStringSlice(amis), testImageId2)
}

func TestAMIGetAll(t *testing.T) {
	t.Parallel()

	testName := "test-ami"
	testImageId := "test-image-id"
	now := time.Now()
	acm := AMIs{
		Client: mockedAMI{
			DescribeImagesOutput: ec2.DescribeImagesOutput{
				Images: []types.Image{{
					ImageId:      &testImageId,
					Name:         &testName,
					CreationDate: aws.String(now.Format("2006-01-02T15:04:05.000Z")),
				}},
			},
		},
	}

	// without filters
	amis, err := acm.getAll(context.Background(), config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(amis), testImageId)

	// with name filter
	amis, err = acm.getAll(context.Background(), config.Config{
		AMI: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-ami"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(amis), testImageId)

	// with time filter
	amis, err = acm.getAll(context.Background(), config.Config{
		AMI: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-12 * time.Hour))}}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(amis), testImageId)
}

func TestAMINukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-ami"
	acm := AMIs{
		Client: mockedAMI{
			DeregisterImageOutput: ec2.DeregisterImageOutput{},
		},
	}

	err := acm.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
