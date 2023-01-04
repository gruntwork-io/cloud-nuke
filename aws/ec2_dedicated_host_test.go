package aws

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TagNamePrefix  = "cloud-nuke-test-"
	InstanceFamily = "c5"
	InstanceType   = "c5.large"
)

func TestListDedicatedHosts(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := ec2.New(session)

	createdHostIds, err := allocateDedicatedHosts(svc, 1)
	require.NoError(t, err)

	defer nukeAllEc2DedicatedHosts(session, createdHostIds)

	// test if created allocation matches get response
	hostIds, err := getAllEc2DedicatedHosts(session, region, time.Now(), config.Config{})

	require.NoError(t, err)
	assert.Equal(t, aws.StringValueSlice(hostIds), aws.StringValueSlice(createdHostIds))

	//test time shift
	olderThan := time.Now().Add(-1 * time.Hour)
	hostIds, err = getAllEc2DedicatedHosts(session, region, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotEqual(t, aws.StringValueSlice(hostIds), aws.StringValueSlice(createdHostIds))
}

func TestNukeDedicatedHosts(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	logging.Logger.Infof("region: %s", region)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := ec2.New(session)

	createdHostIds, err := allocateDedicatedHosts(svc, 1)
	require.NoError(t, err)

	err = nukeAllEc2DedicatedHosts(session, createdHostIds)
	require.NoError(t, err)

	hostIds, err := getAllEc2DedicatedHosts(session, region, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(hostIds), createdHostIds)
}

func allocateDedicatedHosts(svc *ec2.EC2, hostQuantity int64) (hostIds []*string, err error) {

	usableAzs, err := getAzsForInstanceType(svc)

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return nil, gruntworkerrors.WithStackTrace(err)
	}

	if usableAzs == nil {
		logging.Logger.Debugf("[Failed] No AZs with InstanceType found.")
	}

	hostTagName := TagNamePrefix + util.UniqueID()

	input := &ec2.AllocateHostsInput{
		AvailabilityZone: aws.String(usableAzs[0]),
		InstanceFamily:   aws.String(InstanceFamily),
		Quantity:         aws.Int64(hostQuantity),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("dedicated-host"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(hostTagName),
					},
				},
			},
		},
	}

	hostIdsOutput, err := svc.AllocateHosts(input)

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return nil, gruntworkerrors.WithStackTrace(err)
	}

	for i := range hostIdsOutput.HostIds {
		hostIds = append(hostIds, hostIdsOutput.HostIds[i])
	}

	return hostIds, nil
}

func getAzsForInstanceType(svc *ec2.EC2) ([]string, error) {
	var instanceOfferings []string
	input := &ec2.DescribeInstanceTypeOfferingsInput{
		MaxResults:   aws.Int64(5),
		LocationType: aws.String("availability-zone"),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []*string{aws.String(InstanceType)},
			},
		},
	}

	az, err := svc.DescribeInstanceTypeOfferings(input)

	if err != nil {
		return nil, err
	}

	if len(az.InstanceTypeOfferings) == 0 {
		err := errors.New("No matching instance types found in region/az.")
		return nil, gruntworkerrors.WithStackTrace(err)
	}

	for i := range az.InstanceTypeOfferings {
		instanceOfferings = append(instanceOfferings, *az.InstanceTypeOfferings[i].Location)
	}

	return instanceOfferings, err
}
