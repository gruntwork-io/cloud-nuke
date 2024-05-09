package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/efs/efsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedElasticFileSystem struct {
	efsiface.EFSAPI
	DescribeFileSystemsPagesOutput efs.DescribeFileSystemsOutput
	DescribeAccessPointsOutput     efs.DescribeAccessPointsOutput
	DeleteAccessPointOutput        efs.DeleteAccessPointOutput
	DescribeMountTargetsOutput     efs.DescribeMountTargetsOutput
	DeleteMountTargetOutput        efs.DeleteMountTargetOutput
	DeleteFileSystemOutput         efs.DeleteFileSystemOutput
}

func (m mockedElasticFileSystem) DescribeFileSystemsPagesWithContext(_ aws.Context, input *efs.DescribeFileSystemsInput, callback func(*efs.DescribeFileSystemsOutput, bool) bool, _ ...request.Option) error {
	callback(&m.DescribeFileSystemsPagesOutput, true)
	return nil
}

func (m mockedElasticFileSystem) DescribeAccessPointsWithContext(_ aws.Context, input *efs.DescribeAccessPointsInput, _ ...request.Option) (*efs.DescribeAccessPointsOutput, error) {
	return &m.DescribeAccessPointsOutput, nil
}

func (m mockedElasticFileSystem) DeleteAccessPointWithContext(_ aws.Context, input *efs.DeleteAccessPointInput, _ ...request.Option) (*efs.DeleteAccessPointOutput, error) {
	return &m.DeleteAccessPointOutput, nil
}

func (m mockedElasticFileSystem) DescribeMountTargetsWithContext(_ aws.Context, input *efs.DescribeMountTargetsInput, _ ...request.Option) (*efs.DescribeMountTargetsOutput, error) {
	return &m.DescribeMountTargetsOutput, nil
}

func (m mockedElasticFileSystem) DeleteMountTargetWithContext(_ aws.Context, input *efs.DeleteMountTargetInput, _ ...request.Option) (*efs.DeleteMountTargetOutput, error) {
	return &m.DeleteMountTargetOutput, nil
}

func (m mockedElasticFileSystem) DeleteFileSystemWithContext(_ aws.Context, input *efs.DeleteFileSystemInput, _ ...request.Option) (*efs.DeleteFileSystemOutput, error) {
	return &m.DeleteFileSystemOutput, nil
}

func TestEFS_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "testId1"
	testName1 := "test-efs1"
	testId2 := "testId2"
	testName2 := "test-efs2"
	now := time.Now()
	ef := ElasticFileSystem{
		Client: mockedElasticFileSystem{
			DescribeFileSystemsPagesOutput: efs.DescribeFileSystemsOutput{
				FileSystems: []*efs.FileSystemDescription{
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
			names, err := ef.getAll(context.Background(), config.Config{
				ElasticFileSystem: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestEFS_NukeAll(t *testing.T) {

	t.Parallel()

	ef := ElasticFileSystem{
		Client: mockedElasticFileSystem{

			DescribeAccessPointsOutput: efs.DescribeAccessPointsOutput{
				AccessPoints: []*efs.AccessPointDescription{
					{
						AccessPointId: aws.String("fsap-1234567890abcdef0"),
					},
				},
			},
			DescribeMountTargetsOutput: efs.DescribeMountTargetsOutput{
				MountTargets: []*efs.MountTargetDescription{
					{
						MountTargetId: aws.String("fsmt-1234567890abcdef0"),
					},
				},
			},
			DeleteAccessPointOutput: efs.DeleteAccessPointOutput{},
			DeleteMountTargetOutput: efs.DeleteMountTargetOutput{},
			DeleteFileSystemOutput:  efs.DeleteFileSystemOutput{},
		},
	}

	err := ef.nukeAll([]*string{aws.String("fs-1234567890abcdef0")})
	require.NoError(t, err)
}
