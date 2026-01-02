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
	DeleteSnapshotOutput  ec2.DeleteSnapshotOutput
	DeregisterImageOutput ec2.DeregisterImageOutput
	DescribeImagesOutput  ec2.DescribeImagesOutput
	DeletedSnapshots      []string
}

func (m *mockAMIClient) DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error) {
	m.DeletedSnapshots = append(m.DeletedSnapshots, aws.ToString(params.SnapshotId))
	return &m.DeleteSnapshotOutput, nil
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

func TestNukeAMI_DeletesSnapshotsAfterDeregister(t *testing.T) {
	t.Parallel()

	mock := &mockAMIClient{
		DescribeImagesOutput: ec2.DescribeImagesOutput{
			Images: []types.Image{{
				ImageId: aws.String("ami-123"),
				BlockDeviceMappings: []types.BlockDeviceMapping{
					{Ebs: &types.EbsBlockDevice{SnapshotId: aws.String("snap-1")}},
					{Ebs: &types.EbsBlockDevice{SnapshotId: aws.String("snap-2")}},
					{DeviceName: aws.String("/dev/sdf")}, // No EBS - should be skipped
				},
			}},
		},
	}

	err := nukeAMI(context.Background(), mock, aws.String("ami-123"))
	require.NoError(t, err)

	require.Len(t, mock.DeletedSnapshots, 2)
	require.Contains(t, mock.DeletedSnapshots, "snap-1")
	require.Contains(t, mock.DeletedSnapshots, "snap-2")
}

func TestShouldSkipAMI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		image    types.Image
		expected bool
	}{
		{
			name:     "regular image",
			image:    types.Image{Name: aws.String("my-ami")},
			expected: false,
		},
		{
			name:     "AWS Backup image",
			image:    types.Image{Name: aws.String("AwsBackup_daily_123")},
			expected: true,
		},
		{
			name: "AWS managed tag",
			image: types.Image{
				Name: aws.String("regular-name"),
				Tags: []types.Tag{{Key: aws.String("aws-managed"), Value: aws.String("true")}},
			},
			expected: true,
		},
		{
			name: "aws-managed=false should not skip",
			image: types.Image{
				Name: aws.String("regular-name"),
				Tags: []types.Tag{{Key: aws.String("aws-managed"), Value: aws.String("false")}},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := shouldSkipAMI(tc.image)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestGetAMISnapshotIDs(t *testing.T) {
	t.Parallel()

	mock := &mockAMIClient{
		DescribeImagesOutput: ec2.DescribeImagesOutput{
			Images: []types.Image{{
				ImageId: aws.String("ami-123"),
				BlockDeviceMappings: []types.BlockDeviceMapping{
					{Ebs: &types.EbsBlockDevice{SnapshotId: aws.String("snap-abc")}},
					{Ebs: &types.EbsBlockDevice{SnapshotId: aws.String("snap-def")}},
				},
			}},
		},
	}

	ids, err := getAMISnapshotIDs(context.Background(), mock, aws.String("ami-123"))
	require.NoError(t, err)
	require.Len(t, ids, 2)
	require.Equal(t, "snap-abc", aws.ToString(ids[0]))
	require.Equal(t, "snap-def", aws.ToString(ids[1]))
}

func TestGetAMISnapshotIDs_NoImages(t *testing.T) {
	t.Parallel()

	mock := &mockAMIClient{
		DescribeImagesOutput: ec2.DescribeImagesOutput{Images: []types.Image{}},
	}

	ids, err := getAMISnapshotIDs(context.Background(), mock, aws.String("ami-nonexistent"))
	require.NoError(t, err)
	require.Nil(t, ids)
}
