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

func TestSetFirstSeenTag(t *testing.T) {
	t.Parallel()

	const key = "cloud-nuke-first-seen"
	const layout = "2006-01-02 15:04:05"

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	svc := ec2.New(session)
	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	now := time.Now().UTC()

	address := createTestEIPAddress(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllEIPAddresses(session, []*string{address.AllocationId})

	if err := setFirstSeenTag(svc, address, key, now, layout); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	result, err := svc.DescribeTags(&ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("resource-id"),
				Values: []*string{address.AllocationId},
			},
		},
	})

	assert.Len(t, result.Tags, 1)
	assert.Equal(t, key, *result.Tags[0].Key)
	assert.Equal(t, now.Format(layout), *result.Tags[0].Value)
}

func TestGetFirstSeenTag(t *testing.T) {
	t.Parallel()

	const key = "cloud-nuke-first-seen"
	const layout = "2006-01-02 15:04:05"

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	svc := ec2.New(session)
	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	now := time.Now().UTC()

	address := createTestEIPAddress(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllEIPAddresses(session, []*string{address.AllocationId})

	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{address.AllocationId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(key),
				Value: awsgo.String(now.Format(layout)),
			},
		},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	result, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{
		AllocationIds: []*string{address.AllocationId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	firstSeenTime, err := getFirstSeenTag(svc, *result.Addresses[0], key, layout)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	assert.Equal(t, now.Format(layout), (*firstSeenTime).Format(layout))
}

func TestListEIPAddress(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

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

	allocationIds, err := getAllEIPAddresses(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EIP Addresses")
	}

	assert.NotContains(t, awsgo.StringValueSlice(allocationIds), awsgo.StringValue(address.AllocationId))

	allocationIds, err = getAllEIPAddresses(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EIP Addresses")
	}

	assert.Contains(t, awsgo.StringValueSlice(allocationIds), awsgo.StringValue(address.AllocationId))
}

func TestNukeEIPAddress(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

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
