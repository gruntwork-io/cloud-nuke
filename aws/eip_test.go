package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestEIPAddress(t *testing.T, session *session.Session, name string) ec2.Address {
	svc := ec2.New(session)
	result, err := svc.AllocateAddress(&ec2.AllocateAddressInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	output, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{
		AllocationIds: []*string{result.AllocationId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return *output.Addresses[0]
}

func TestListEIPAddress(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()

	address := createTestEIPAddress(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllEIPAddresses(session, []*string{address.AllocationId})

	allocationIds, err := getAllEIPAddresses(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EIP Addresses")
	}

	assert.Contains(t, awsgo.StringValueSlice(allocationIds), awsgo.StringValue(address.AllocationId))
}

func TestNukeEIPAddress(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	address := createTestEIPAddress(t, session, uniqueTestID)

	if err := nukeAllEIPAddresses(session, []*string{address.AllocationId}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	allocationIds, err := getAllEIPAddresses(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EIP Addresses")
	}

	assert.NotContains(t, awsgo.StringValueSlice(allocationIds), awsgo.StringValue(address.AllocationId))
}
