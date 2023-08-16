package resources

import (
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"

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
	amis, err := acm.getAll(config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(amis), testImageId)

	// with name filter
	amis, err = acm.getAll(config.Config{
		AMI: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-ami"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(amis), testImageId)

	// with time filter
	amis, err = acm.getAll(config.Config{
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
