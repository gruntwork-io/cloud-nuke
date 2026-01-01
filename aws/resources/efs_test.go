package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedElasticFileSystem struct {
	DeleteAccessPointOutput    efs.DeleteAccessPointOutput
	DeleteFileSystemOutput     efs.DeleteFileSystemOutput
	DeleteMountTargetOutput    efs.DeleteMountTargetOutput
	DescribeAccessPointsOutput efs.DescribeAccessPointsOutput
	DescribeMountTargetsOutput efs.DescribeMountTargetsOutput
	DescribeFileSystemsOutput  efs.DescribeFileSystemsOutput

	// Track calls to DescribeMountTargets to simulate deletion
	describeMountTargetsCalls int
}

func (m mockedElasticFileSystem) DeleteAccessPoint(ctx context.Context, params *efs.DeleteAccessPointInput, optFns ...func(*efs.Options)) (*efs.DeleteAccessPointOutput, error) {
	return &m.DeleteAccessPointOutput, nil
}

func (m mockedElasticFileSystem) DeleteFileSystem(ctx context.Context, params *efs.DeleteFileSystemInput, optFns ...func(*efs.Options)) (*efs.DeleteFileSystemOutput, error) {
	return &m.DeleteFileSystemOutput, nil
}

func (m mockedElasticFileSystem) DeleteMountTarget(ctx context.Context, params *efs.DeleteMountTargetInput, optFns ...func(*efs.Options)) (*efs.DeleteMountTargetOutput, error) {
	return &m.DeleteMountTargetOutput, nil
}

func (m mockedElasticFileSystem) DescribeAccessPoints(ctx context.Context, params *efs.DescribeAccessPointsInput, optFns ...func(*efs.Options)) (*efs.DescribeAccessPointsOutput, error) {
	return &m.DescribeAccessPointsOutput, nil
}

func (m *mockedElasticFileSystem) DescribeMountTargets(ctx context.Context, params *efs.DescribeMountTargetsInput, optFns ...func(*efs.Options)) (*efs.DescribeMountTargetsOutput, error) {
	m.describeMountTargetsCalls++
	// First call returns mount targets (used during enumeration and first waiter check)
	// Subsequent calls return empty list (simulating mount targets being deleted)
	if m.describeMountTargetsCalls <= 1 {
		return &m.DescribeMountTargetsOutput, nil
	}
	return &efs.DescribeMountTargetsOutput{MountTargets: []types.MountTargetDescription{}}, nil
}

func (m mockedElasticFileSystem) DescribeFileSystems(ctx context.Context, params *efs.DescribeFileSystemsInput, optFns ...func(*efs.Options)) (*efs.DescribeFileSystemsOutput, error) {
	return &m.DescribeFileSystemsOutput, nil
}

func TestEFS_GetAll(t *testing.T) {
	t.Parallel()
	testId1 := "testId1"
	testName1 := "test-efs1"
	testId2 := "testId2"
	testName2 := "test-efs2"
	now := time.Now()
	client := &mockedElasticFileSystem{
		DescribeFileSystemsOutput: efs.DescribeFileSystemsOutput{
			FileSystems: []types.FileSystemDescription{
				{
					FileSystemId: aws.String(testId1),
					Name:         aws.String(testName1),
					CreationTime: aws.Time(now),
				},
				{
					FileSystemId: aws.String(testId2),
					Name:         aws.String(testName2),
					CreationTime: aws.Time(now.Add(1)),
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listElasticFileSystems(context.Background(), client, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEFS_NukeAll(t *testing.T) {
	t.Parallel()
	client := &mockedElasticFileSystem{
		DescribeAccessPointsOutput: efs.DescribeAccessPointsOutput{
			AccessPoints: []types.AccessPointDescription{
				{
					AccessPointId: aws.String("fsap-1234567890abcdef0"),
				},
			},
		},
		DescribeMountTargetsOutput: efs.DescribeMountTargetsOutput{
			MountTargets: []types.MountTargetDescription{
				{
					MountTargetId: aws.String("fsmt-1234567890abcdef0"),
				},
			},
		},
		DeleteAccessPointOutput: efs.DeleteAccessPointOutput{},
		DeleteMountTargetOutput: efs.DeleteMountTargetOutput{},
		DeleteFileSystemOutput:  efs.DeleteFileSystemOutput{},
	}

	err := deleteElasticFileSystems(context.Background(), client, resource.Scope{Region: "us-east-1"}, "efs", []*string{aws.String("fs-1234567890abcdef0")})
	require.NoError(t, err)
}
