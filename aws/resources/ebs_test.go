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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEBSVolumesClient struct {
	DescribeVolumesOutput ec2.DescribeVolumesOutput
	DeleteVolumeOutput    ec2.DeleteVolumeOutput
}

func (m *mockEBSVolumesClient) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return &m.DescribeVolumesOutput, nil
}

func (m *mockEBSVolumesClient) DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error) {
	return &m.DeleteVolumeOutput, nil
}

func TestEBSVolumes_ResourceName(t *testing.T) {
	r := NewEBSVolumes()
	assert.Equal(t, "ebs", r.ResourceName())
}

func TestEBSVolumes_MaxBatchSize(t *testing.T) {
	r := NewEBSVolumes()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListEBSVolumes(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testVolume1 := "test-volume1"
	testVolume2 := "test-volume2"
	now := time.Now()

	mock := &mockEBSVolumesClient{
		DescribeVolumesOutput: ec2.DescribeVolumesOutput{
			Volumes: []types.Volume{
				{
					VolumeId:   aws.String(testVolume1),
					CreateTime: aws.Time(now),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String(testName1),
					}},
				},
				{
					VolumeId:   aws.String(testVolume2),
					CreateTime: aws.Time(now.Add(1)),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String(testName2),
					}},
				},
			},
		},
	}

	ids, err := listEBSVolumes(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testVolume1, testVolume2}, aws.ToStringSlice(ids))
}

func TestListEBSVolumes_WithNameExclusionFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testVolume1 := "test-volume1"
	testVolume2 := "test-volume2"
	now := time.Now()

	mock := &mockEBSVolumesClient{
		DescribeVolumesOutput: ec2.DescribeVolumesOutput{
			Volumes: []types.Volume{
				{
					VolumeId:   aws.String(testVolume1),
					CreateTime: aws.Time(now),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String(testName1),
					}},
				},
				{
					VolumeId:   aws.String(testVolume2),
					CreateTime: aws.Time(now.Add(1)),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String(testName2),
					}},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
		},
	}

	ids, err := listEBSVolumes(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testVolume2}, aws.ToStringSlice(ids))
}

func TestListEBSVolumes_WithTimeAfterExclusionFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testVolume1 := "test-volume1"
	testVolume2 := "test-volume2"
	now := time.Now()

	mock := &mockEBSVolumesClient{
		DescribeVolumesOutput: ec2.DescribeVolumesOutput{
			Volumes: []types.Volume{
				{
					VolumeId:   aws.String(testVolume1),
					CreateTime: aws.Time(now),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String(testName1),
					}},
				},
				{
					VolumeId:   aws.String(testVolume2),
					CreateTime: aws.Time(now.Add(1)),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String(testName2),
					}},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-2 * time.Hour)),
		},
	}

	ids, err := listEBSVolumes(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestDeleteEBSVolume(t *testing.T) {
	t.Parallel()

	mock := &mockEBSVolumesClient{
		DeleteVolumeOutput: ec2.DeleteVolumeOutput{},
	}

	err := deleteEBSVolume(context.Background(), mock, aws.String("test-volume"))
	require.NoError(t, err)
}
