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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockNetworkACLClient struct {
	DescribeNetworkAclsOutput          ec2.DescribeNetworkAclsOutput
	DeleteNetworkAclOutput             ec2.DeleteNetworkAclOutput
	ReplaceNetworkAclAssociationOutput ec2.ReplaceNetworkAclAssociationOutput
}

func (m *mockNetworkACLClient) DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
	return &m.DescribeNetworkAclsOutput, nil
}

func (m *mockNetworkACLClient) DeleteNetworkAcl(ctx context.Context, params *ec2.DeleteNetworkAclInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkAclOutput, error) {
	return &m.DeleteNetworkAclOutput, nil
}

func (m *mockNetworkACLClient) ReplaceNetworkAclAssociation(ctx context.Context, params *ec2.ReplaceNetworkAclAssociationInput, optFns ...func(*ec2.Options)) (*ec2.ReplaceNetworkAclAssociationOutput, error) {
	return &m.ReplaceNetworkAclAssociationOutput, nil
}

func TestListNetworkACLs(t *testing.T) {
	t.Parallel()

	testId1 := "acl-09e36c45cbdbfb001"
	testId2 := "acl-09e36c45cbdbfb002"
	testName1 := "cloud-nuke-acl-001"
	testName2 := "cloud-nuke-acl-002"
	now := time.Now()

	mock := &mockNetworkACLClient{
		DescribeNetworkAclsOutput: ec2.DescribeNetworkAclsOutput{
			NetworkAcls: []types.NetworkAcl{
				{
					NetworkAclId: aws.String(testId1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					NetworkAclId: aws.String(testId2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId2},
		},
		"nameInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{testId1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
			ids, err := listNetworkACLs(ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestReplaceNetworkACLAssociations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		describeOutput ec2.DescribeNetworkAclsOutput
		expectError    bool
	}{
		"withAssociations": {
			describeOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{
					{
						NetworkAclId: aws.String("acl-001"),
						VpcId:        aws.String("vpc-123"),
						Associations: []types.NetworkAclAssociation{
							{
								NetworkAclAssociationId: aws.String("aclassoc-001"),
								NetworkAclId:            aws.String("acl-001"),
								SubnetId:                aws.String("subnet-1234"),
							},
						},
					},
				},
			},
			expectError: false,
		},
		"noAssociations": {
			describeOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{
					{
						NetworkAclId: aws.String("acl-002"),
						VpcId:        aws.String("vpc-456"),
						Associations: []types.NetworkAclAssociation{},
					},
				},
			},
			expectError: false,
		},
		"notFound": {
			describeOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{},
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockNetworkACLClient{
				DescribeNetworkAclsOutput: tc.describeOutput,
			}

			err := replaceNetworkACLAssociations(context.Background(), mock, aws.String("acl-001"))

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeleteNetworkACL(t *testing.T) {
	t.Parallel()

	mock := &mockNetworkACLClient{
		DeleteNetworkAclOutput: ec2.DeleteNetworkAclOutput{},
	}

	err := deleteNetworkACL(context.Background(), mock, aws.String("acl-001"))
	require.NoError(t, err)
}
