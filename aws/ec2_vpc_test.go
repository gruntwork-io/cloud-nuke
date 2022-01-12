package aws

import (
	"regexp"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestVpc(t *testing.T, session *session.Session) string {
	svc := ec2.New(session)
	vpc, err := svc.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: awsgo.String("10.0.0.0/24"),
	})

	require.NoError(t, err)

	err = svc.WaitUntilVpcExists(&ec2.DescribeVpcsInput{
		VpcIds: awsgo.StringSlice([]string{*vpc.Vpc.VpcId}),
	})

	require.NoError(t, err)
	return *vpc.Vpc.VpcId
}

func TestListVpcs(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	vpcId := createTestVpc(t, session)

	// clean up after this test
	defer nukeAllVPCs(session, []string{vpcId}, []Vpc{{
		Region: region,
		VpcId:  vpcId,
		svc:    ec2.New(session),
	}})

	vpcIds, _, err := getAllVpcs(session, region, config.Config{})
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(vpcIds), vpcId)
}

func TestListVpcsWithConfigFile(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	includedVpcId := createTestVpc(t, session)
	excludedVpcId := createTestVpc(t, session)

	// clean up after this test
	defer nukeAllVPCs(session, []string{includedVpcId, excludedVpcId}, []Vpc{{
		Region: region,
		VpcId:  includedVpcId,
		svc:    ec2.New(session),
	}, {
		Region: region,
		VpcId:  excludedVpcId,
		svc:    ec2.New(session),
	}})

	vpcIds, _, err := getAllVpcs(session, region, config.Config{
		VPC: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile(includedVpcId)},
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, len(vpcIds))
	assert.Contains(t, awsgo.StringValueSlice(vpcIds), includedVpcId)
}

func TestNukeVpcs(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	vpcId := createTestVpc(t, session)

	// clean up after this test
	err = nukeAllVPCs(session, []string{vpcId}, []Vpc{{
		Region: region,
		VpcId:  vpcId,
		svc:    ec2.New(session),
	}})

	require.NoError(t, err)

	vpcIds, _, err := getAllVpcs(session, region, config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(vpcIds), vpcId)
}
