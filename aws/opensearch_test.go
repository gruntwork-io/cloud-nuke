package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/opensearchservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Excluded regions which doesn't include "t3.small.search" instances
var ExcludedOpenSearchDomains = []string{
	"ap-northeast-3",
}

// Test we can create an OpenSearch Domain, tag it, and then find the tag
func TestCanTagOpenSearchDomains(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegionWithExclusions(ExcludedOpenSearchDomains)
	require.NoError(t, err)

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	domain := createOpenSearchDomain(t, awsSession, region, newOpenSearchDomainName())
	defer deleteOpenSearchDomain(t, awsSession, domain.DomainName, true)

	tagValue := time.Now().UTC()

	tagErr := tagOpenSearchDomainWhenFirstSeen(awsSession, domain.ARN, tagValue)
	require.NoError(t, tagErr)

	returnedTag, err := getFirstSeenOpenSearchDomainTag(awsSession, domain.ARN)
	require.NoError(t, err)

	parsedTagValue, parseErr1 := parseTimestampTag(formatTimestampTag(tagValue))
	require.NoError(t, parseErr1)

	parsedReturnValue, parseErr2 := parseTimestampTag(formatTimestampTag(returnedTag))
	require.NoError(t, parseErr2)

	// compare that the tags' Time values after formatting are equal
	assert.Equal(t, parsedTagValue, parsedReturnValue)
}

// Test we can get all OpenSearch domains younger than < X time based on tags
func TestCanListAllOpenSearchDomainsOlderThan24hours(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegionWithExclusions(ExcludedOpenSearchDomains)
	require.NoError(t, err)

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	domain1 := createOpenSearchDomain(t, awsSession, region, newOpenSearchDomainName())
	defer deleteOpenSearchDomain(t, awsSession, domain1.DomainName, true)
	domain2 := createOpenSearchDomain(t, awsSession, region, newOpenSearchDomainName())
	defer deleteOpenSearchDomain(t, awsSession, domain2.DomainName, true)

	now := time.Now().UTC()
	olderClusterTagValue := now.Add(time.Hour * time.Duration(-48))
	youngerClusterTagValue := now.Add(time.Hour * time.Duration(-23))

	err1 := tagOpenSearchDomainWhenFirstSeen(awsSession, domain1.ARN, olderClusterTagValue)
	require.NoError(t, err1)
	err2 := tagOpenSearchDomainWhenFirstSeen(awsSession, domain2.ARN, youngerClusterTagValue)
	require.NoError(t, err2)

	last24Hours := now.Add(time.Hour * time.Duration(-24))
	domains, err := getOpenSearchDomainsToNuke(awsSession, last24Hours, config.Config{})
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(domains), awsgo.StringValue(domain1.DomainName))
	assert.NotContains(t, awsgo.StringValueSlice(domains), awsgo.StringValue(domain2.DomainName))
}

// Test we can nuke OpenSearch Domains
func TestCanNukeOpenSearchDomain(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegionWithExclusions(ExcludedOpenSearchDomains)
	require.NoError(t, err)

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	domain := createOpenSearchDomain(t, awsSession, region, newOpenSearchDomainName())
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	defer deleteOpenSearchDomain(t, awsSession, domain.DomainName, false)

	require.NoError(t, nukeAllOpenSearchDomains(awsSession, []*string{domain.DomainName}))

	allLeftDomains, err := getAllActiveOpenSearchDomains(awsSession)
	require.NoError(t, err)
	allLeftDomainNames := []*string{}
	for _, domain := range allLeftDomains {
		allLeftDomainNames = append(allLeftDomainNames, domain.DomainName)
	}
	assert.NotContains(t, awsgo.StringValueSlice(allLeftDomainNames), awsgo.StringValue(domain.DomainName))
}

// Helper functions for driving the OpenSearch Domains tests

// createOpenSearchDomain will create a new OpenSearch Domain in the default VPC
func createOpenSearchDomain(t *testing.T, awsSession *session.Session, region string, name string) *opensearchservice.DomainStatus {
	svc := opensearchservice.New(awsSession)
	resp, err := svc.CreateDomain(&opensearchservice.CreateDomainInput{
		DomainName:    awsgo.String(name),
		ClusterConfig: &opensearchservice.ClusterConfig{InstanceType: awsgo.String("t3.small.search")},
		EBSOptions: &opensearchservice.EBSOptions{
			EBSEnabled: awsgo.Bool(true),
			VolumeSize: awsgo.Int64(10),
		},
	})
	require.NoError(t, err)
	return resp.DomainStatus
}

// deleteOpenSearchDomain is a function to delete the given OpenSearch Domain.
func deleteOpenSearchDomain(t *testing.T, awsSession *session.Session, domainName *string, checkErr bool) {
	svc := opensearchservice.New(awsSession)
	input := &opensearchservice.DeleteDomainInput{DomainName: domainName}
	_, err := svc.DeleteDomain(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func newOpenSearchDomainName() string {
	return "cloud-nuke-test-" + strings.ToLower(util.UniqueID())
}
