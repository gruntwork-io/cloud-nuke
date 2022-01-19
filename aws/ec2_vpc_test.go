package aws

import (
	"regexp"
	"testing"
	"time"

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

func TestCanTagVpc(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	vpcId := createTestVpc(t, session)
	svc := ec2.New(session)

	// clean up after this test
	defer nukeAllVPCs(session, []string{vpcId}, []Vpc{{
		Region: region,
		VpcId:  vpcId,
		svc:    svc,
	}})

	result, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: awsgo.StringSlice([]string{vpcId}),
	})

	require.NoError(t, err)
	require.Equal(t, 1, len(result.Vpcs))

	vpc := result.Vpcs[0]
	key := "cloud-nuke-first-seen-test"
	value := time.Now().UTC()

	err = setFirstSeenVpcTag(svc, *vpc, key, value)
	require.NoError(t, err)

	result, err = svc.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: awsgo.StringSlice([]string{vpcId}),
	})

	require.NoError(t, err)

	vpc = result.Vpcs[0]
	value1, err := getFirstSeenVpcTag(*vpc, key)
	require.NoError(t, err)
	require.NotNil(t, value1)

	// Parsing from string doesn't include the millisecond,
	// so format the dates according to this layout so we can
	// perform a direct comparison.
	layout := "2006-01-02T15:04:05"
	assert.Equal(t, value.Format(layout), (*value1).Format(layout))
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

	// First run gives us a chance to tag the VPC
	_, _, err = getAllVpcs(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	// VPC should be tagged at this point
	vpcIds, _, err := getAllVpcs(session, region, time.Now().Add(1*time.Hour), config.Config{})
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

	// First run gives us a chance to tag the VPC
	_, _, err = getAllVpcs(session, region, time.Now().Add(1*time.Hour), config.Config{
		VPC: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile(includedVpcId)},
				},
			},
		},
	})

	require.NoError(t, err)

	// VPC should be tagged at this point
	vpcIds, _, err := getAllVpcs(session, region, time.Now().Add(1*time.Hour), config.Config{
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

	// First run gives us a chance to tag the VPC
	_, _, err = getAllVpcs(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	// VPC should be tagged at this point
	vpcIds, _, err := getAllVpcs(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(vpcIds), vpcId)
}
