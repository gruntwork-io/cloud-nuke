package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestRDSCluster(t *testing.T, session *session.Session, name string) {
	t.Logf("Creating RDS Cluster in region %s", aws.StringValue(session.Config.Region))

	svc := rds.New(session)
	params := &rds.CreateDBClusterInput{
		DBClusterIdentifier: awsgo.String(name),
		Engine:              awsgo.String("aurora-mysql"),
		MasterUsername:      awsgo.String("gruntwork"),
		MasterUserPassword:  awsgo.String("password"),
	}

	_, err := svc.CreateDBCluster(params)
	require.NoError(t, err)
}

func TestNukeRDSCluster(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	rdsName := "cloud-nuke-test" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestRDSCluster(t, session, rdsName)

	defer func() {
		nukeAllRdsClusters(session, []*string{&rdsName})

		rdsNames, _ := getAllRdsClusters(session, excludeAfter)

		assert.NotContains(t, awsgo.StringValueSlice(rdsNames), strings.ToLower(rdsName))
	}()

	rds, err := getAllRdsClusters(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Clusters", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(rds), strings.ToLower(rdsName))
}
