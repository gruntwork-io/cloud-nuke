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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockAMIClient struct {
	DeregisterImageOutput ec2.DeregisterImageOutput
	DescribeImagesOutput  ec2.DescribeImagesOutput
}

func (m *mockAMIClient) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	return &m.DeregisterImageOutput, nil
}

func (m *mockAMIClient) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOutput, nil
}

func TestListAMIs_SkipAWSManaged(t *testing.T) {
	t.Parallel()

	testName := "test-ami"
	testImageId1 := "test-image-id1"
	testImageId2 := "test-image-id2"
	now := time.Now()
	mock := &mockAMIClient{
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
	}

	amis, err := listAMIs(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.NotContains(t, aws.ToStringSlice(amis), testImageId1)
	require.NotContains(t, aws.ToStringSlice(amis), testImageId2)
}

func TestListAMIs(t *testing.T) {
	t.Parallel()

	testName := "test-ami"
	testImageId := "test-image-id"
	now := time.Now()
	mock := &mockAMIClient{
		DescribeImagesOutput: ec2.DescribeImagesOutput{
			Images: []types.Image{{
				ImageId:      &testImageId,
				Name:         &testName,
				CreationDate: aws.String(now.Format("2006-01-02T15:04:05.000Z")),
			}},
		},
	}

	// without filters
	amis, err := listAMIs(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(amis), testImageId)

	// with name filter
	amis, err = listAMIs(context.Background(), mock, resource.Scope{}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{
				RE: *regexp.MustCompile("test-ami"),
			}},
		},
	})
	require.NoError(t, err)
	require.NotContains(t, aws.ToStringSlice(amis), testImageId)

	// with time filter
	amis, err = listAMIs(context.Background(), mock, resource.Scope{}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-12 * time.Hour)),
		},
	})
	require.NoError(t, err)
	require.NotContains(t, aws.ToStringSlice(amis), testImageId)
}

func TestDeleteAMI(t *testing.T) {
	t.Parallel()

	mock := &mockAMIClient{}
	err := deleteAMI(context.Background(), mock, aws.String("test-image-id"))
	require.NoError(t, err)
}
