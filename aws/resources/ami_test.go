package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type mockedAMI struct {
	ec2iface.EC2API
	DescribeImagesOutput  ec2.DescribeImagesOutput
	DeregisterImageOutput ec2.DeregisterImageOutput
}

func (m mockedAMI) DescribeImages(input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOutput, nil
}

func (m mockedAMI) DeregisterImage(input *ec2.DeregisterImageInput) (*ec2.DeregisterImageOutput, error) {
	return &m.DeregisterImageOutput, nil
}

func TestAMIGetAll_SkipAWSManaged(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName := "test-ami"
	testImageId1 := "test-image-id1"
	testImageId2 := "test-image-id2"
	now := time.Now()
	acm := AMIs{
		Client: mockedAMI{
			DescribeImagesOutput: ec2.DescribeImagesOutput{
				Images: []*ec2.Image{
					{
						ImageId:      &testImageId1,
						Name:         &testName,
						CreationDate: awsgo.String(now.Format("2006-01-02T15:04:05.000Z")),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("aws-managed"),
								Value: aws.String("true"),
							},
						},
					},
					{
						ImageId:      &testImageId2,
						Name:         aws.String("AwsBackup_Test"),
						CreationDate: awsgo.String(now.Format("2006-01-02T15:04:05.000Z")),
						Tags: []*ec2.Tag{
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
	assert.NotContains(t, awsgo.StringValueSlice(amis), testImageId1)
	assert.NotContains(t, awsgo.StringValueSlice(amis), testImageId2)
}

func TestAMIGetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName := "test-ami"
	testImageId := "test-image-id"
	now := time.Now()
	acm := AMIs{
		Client: mockedAMI{
			DescribeImagesOutput: ec2.DescribeImagesOutput{
				Images: []*ec2.Image{{
					ImageId:      &testImageId,
					Name:         &testName,
					CreationDate: awsgo.String(now.Format("2006-01-02T15:04:05.000Z")),
				}},
			},
		},
	}

	// without filters
	amis, err := acm.getAll(context.Background(), config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(amis), testImageId)

	// with name filter
	amis, err = acm.getAll(context.Background(), config.Config{
		AMI: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-ami"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(amis), testImageId)

	// with time filter
	amis, err = acm.getAll(context.Background(), config.Config{
		AMI: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: awsgo.Time(now.Add(-12 * time.Hour))}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(amis), testImageId)
}

func TestAMINukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
