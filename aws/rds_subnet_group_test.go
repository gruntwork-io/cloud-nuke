package aws

import (
	"fmt"
	"strings"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestRDSSubnetGroup(t *testing.T, session *session.Session, name string) {
	t.Logf("Creating RDS subnet group in region %s", awsgo.StringValue(session.Config.Region))

	defaultVpc := aws.GetDefaultVpc(t, *session.Config.Region)
	defaultAzSubnets := aws.GetDefaultSubnetIDsForVpc(t, *defaultVpc)
	var subnetIds []*string
	for _, subnet := range defaultAzSubnets {
		subnetIds = append(subnetIds, awsgo.String(subnet))
	}

	svc := rds.New(session)
	_, err := svc.CreateDBSubnetGroup(&rds.CreateDBSubnetGroupInput{
		DBSubnetGroupName:        awsgo.String(name),
		DBSubnetGroupDescription: awsgo.String(fmt.Sprintf("Test DB subnet for %s", t.Name())),
		SubnetIds:                subnetIds,
	})

	require.NoError(t, err)
}

func TestNukeRDSSubnetGroup(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	subnetGroupName := "cloud-nuke-test-" + util.UniqueID()
	createTestRDSSubnetGroup(t, session, subnetGroupName)

	defer func() {
		nukeAllRdsDbSubnetGroups(session, []*string{&subnetGroupName})
		subnetGroupNames, _ := getAllRdsDbSubnetGroups(session, config.Config{})
		assert.NotContains(t, awsgo.StringValueSlice(subnetGroupNames), strings.ToLower(subnetGroupName))
	}()
}
