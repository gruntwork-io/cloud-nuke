package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkACL struct {
	DescribeNetworkAclsOutput          ec2.DescribeNetworkAclsOutput
	DeleteNetworkAclOutput             ec2.DeleteNetworkAclOutput
	ReplaceNetworkAclAssociationOutput ec2.ReplaceNetworkAclAssociationOutput
}

func (m *mockedNetworkACL) DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
	return &m.DescribeNetworkAclsOutput, nil
}

func (m *mockedNetworkACL) DeleteNetworkAcl(ctx context.Context, params *ec2.DeleteNetworkAclInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkAclOutput, error) {
	return &m.DeleteNetworkAclOutput, nil
}

func (m *mockedNetworkACL) ReplaceNetworkAclAssociation(ctx context.Context, params *ec2.ReplaceNetworkAclAssociationInput, optFns ...func(*ec2.Options)) (*ec2.ReplaceNetworkAclAssociationOutput, error) {
	return &m.ReplaceNetworkAclAssociationOutput, nil
}

func TestNetworkAcl_GetAll(t *testing.T) {

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now     = time.Now()
		testId1 = awsgo.String("acl-09e36c45cbdbfb001")
		testId2 = awsgo.String("acl-09e36c45cbdbfb002")

		testName1 = "cloud-nuke-acl-001"
		testName2 = "cloud-nuke-acl-002"
	)

	resourceObject := NetworkACL{
		Client: &mockedNetworkACL{
			DescribeNetworkAclsOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{
					{
						NetworkAclId: testId1,
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						NetworkAclId: testId2,
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []*string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []*string{testId1, testId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []*string{testId2},
		},
		"nameInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []*string{testId1},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: awsgo.Time(now),
				}},
			expected: []*string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := resourceObject.getAll(tc.ctx, config.Config{
				NetworkACL: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, names)
		})
	}
}

func TestNetworkAcl_NukeAll(t *testing.T) {
	var (
		testId1 = "acl-09e36c45cbdbfb001"
		testId2 = "acl-09e36c45cbdbfb002"

		testName1 = "cloud-nuke-acl-001"
		testName2 = "cloud-nuke-acl-002"
	)

	resourceObject := NetworkACL{
		Client: &mockedNetworkACL{
			DescribeNetworkAclsOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{
					{
						NetworkAclId: awsgo.String(testId1),
						Associations: []types.NetworkAclAssociation{
							{
								NetworkAclAssociationId: awsgo.String("assoc-09e36c45cbdbfb001"),
								NetworkAclId:            awsgo.String(testId1),
								SubnetId:                awsgo.String("subnet-1234"),
							},
						},
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
						},
					},
					{
						NetworkAclId: awsgo.String(testId2),
						Associations: []types.NetworkAclAssociation{
							{
								NetworkAclAssociationId: awsgo.String("assoc-09e36c45cbdbfb002"),
								NetworkAclId:            awsgo.String(testId2),
								SubnetId:                awsgo.String("subnet-5678"),
							},
						},
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
						},
					},
				},
			},
		},
	}
	err := resourceObject.nukeAll([]*string{
		awsgo.String(testId1),
		awsgo.String(testId2),
	})
	require.NoError(t, err)
}
