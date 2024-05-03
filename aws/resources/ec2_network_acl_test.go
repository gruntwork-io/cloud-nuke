package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkACL struct {
	BaseAwsResource
	ec2iface.EC2API

	DescribeNetworkAclsOutput          ec2.DescribeNetworkAclsOutput
	DeleteNetworkAclOutput             ec2.DeleteNetworkAclOutput
	ReplaceNetworkAclAssociationOutput ec2.ReplaceNetworkAclAssociationOutput
}

func (m mockedNetworkACL) DescribeNetworkAcls(*ec2.DescribeNetworkAclsInput) (*ec2.DescribeNetworkAclsOutput, error) {
	return &m.DescribeNetworkAclsOutput, nil
}

func (m mockedNetworkACL) DeleteNetworkAcl(*ec2.DeleteNetworkAclInput) (*ec2.DeleteNetworkAclOutput, error) {
	return &m.DeleteNetworkAclOutput, nil
}

func (m mockedNetworkACL) ReplaceNetworkAclAssociation(*ec2.ReplaceNetworkAclAssociationInput) (*ec2.ReplaceNetworkAclAssociationOutput, error) {
	return &m.ReplaceNetworkAclAssociationOutput, nil
}

func TestNetworkAcl_GetAll(t *testing.T) {

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now     = time.Now()
		testId1 = "acl-09e36c45cbdbfb001"
		testId2 = "acl-09e36c45cbdbfb002"

		testName1 = "cloud-nuke-acl-001"
		testName2 = "cloud-nuke-acl-002"
	)

	resourceObject := NetworkACL{
		Client: mockedNetworkACL{
			DescribeNetworkAclsOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []*ec2.NetworkAcl{
					{
						NetworkAclId: aws.String(testId1),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						NetworkAclId: aws.String(testId2),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
			},
		},
	}
	resourceObject.BaseAwsResource.Init(nil)

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"nameInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId1},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{
				testId1,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := resourceObject.getAll(tc.ctx, config.Config{
				NetworkACL: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
		BaseAwsResource: BaseAwsResource{
			Nukables: map[string]error{
				testId1: nil,
				testId2: nil,
			},
		},
		Client: mockedNetworkACL{
			DescribeNetworkAclsOutput: ec2.DescribeNetworkAclsOutput{
				NetworkAcls: []*ec2.NetworkAcl{
					{
						NetworkAclId: aws.String(testId1),
						Associations: []*ec2.NetworkAclAssociation{
							{
								NetworkAclAssociationId: aws.String("assoc-09e36c45cbdbfb001"),
								NetworkAclId:            aws.String("acl-09e36c45cbdbfb001"),
							},
						},
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
						},
					},
					{
						NetworkAclId: aws.String(testId2),
						Associations: []*ec2.NetworkAclAssociation{
							{
								NetworkAclAssociationId: aws.String("assoc-09e36c45cbdbfb002"),
								NetworkAclId:            aws.String("acl-09e36c45cbdbfb002"),
							},
						},
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
						},
					},
				},
			},
		},
	}

	err := resourceObject.nukeAll([]*string{
		aws.String(testId1),
		aws.String(testId2),
	})
	require.NoError(t, err)
}
