package aws

import (
	"context"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestElasticFileSystem struct {
	Name *string
	ID   *string
}

func createTestElasticFileSystem(t *testing.T, session *session.Session, name string) (*TestElasticFileSystem, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	svc := efs.NewFromConfig(cfg)

	testEfs := &TestElasticFileSystem{
		Name: aws.String(name),
	}

	param := &efs.CreateFileSystemInput{
		CreationToken: testEfs.Name,
	}

	output, err := svc.CreateFileSystem(context.TODO(), param)
	if err != nil {
		assert.Failf(t, "Could not create test Elastic FileSystem (efs): %s", errors.WithStackTrace(err).Error())
	}

	testEfs.ID = output.FileSystemId

	return testEfs, nil
}

func TestListElasticFileSystems(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
	)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	efsName := "aws-nuke-test-" + util.UniqueID()
	testEfs, createTestEfsErr := createTestElasticFileSystem(t, session, efsName)
	require.NoError(t, createTestEfsErr)
	// clean up after this test
	defer nukeAllElasticFileSystems(session, []*string{testEfs.ID})

	efsIds, err := getAllElasticFileSystems(session, time.Now(), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Elastic FileSystems (efs)")
	}

	assert.Contains(t, awsgo.StringValueSlice(efsIds), aws.StringValue(testEfs.ID))
}

func TestTimeFilterExclusionNewlyCreatedEFS(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	testEFSName := "aws-nuke-test-" + util.UniqueID()

	testEFS, createEFSErr := createTestElasticFileSystem(t, session, testEFSName)
	require.NoError(t, createEFSErr)
	defer nukeAllElasticFileSystems(session, []*string{testEFS.ID})

	// Assert Elastic FileSystem is picked up without filters
	efsIds, err := getAllElasticFileSystems(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(efsIds), aws.StringValue(testEFS.ID))

	// Assert Elastic FileSystem doesn't appear when we look at Elastic FileSystems older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	efsIdsOlder, err := getAllElasticFileSystems(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(efsIdsOlder), aws.StringValue(testEFS.ID))
}

func TestNukeElasticFileSystemOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testEfs, createTestErr := createTestElasticFileSystem(t, session, apigwName)
	require.NoError(t, createTestErr)

	nukeErr := nukeAllElasticFileSystems(session, []*string{testEfs.ID})
	require.NoError(t, nukeErr)

	// This sleep is necessary to allow AWS to realize the Elastic FileSystem is no longer "in-use"
	time.Sleep(10 * time.Second)

	// Make sure the Elastic FileSystem was deleted
	efsIds, err := getAllElasticFileSystems(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(efsIds), aws.StringValue(testEfs.ID))
}

func TestNukeElasticFileSystemsMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	efsName := "aws-nuke-test-" + util.UniqueID()
	efsName2 := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testEFS, createTestErr := createTestElasticFileSystem(t, session, efsName)
	require.NoError(t, createTestErr)
	testEFS2, createTestErr2 := createTestElasticFileSystem(t, session, efsName2)
	require.NoError(t, createTestErr2)

	nukeErr := nukeAllElasticFileSystems(session, []*string{testEFS.ID, testEFS2.ID})
	require.NoError(t, nukeErr)

	// Sleep for 10 seconds so that AWS has time to realize the EFS is no longer "in-use"
	time.Sleep(10 * time.Second)

	// Make sure the Elastic FileSystems were deleted
	efsIds, err := getAllElasticFileSystems(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(efsIds), aws.StringValue(testEFS.ID))
	assert.NotContains(t, aws.StringValueSlice(efsIds), aws.StringValue(testEFS2.ID))
}
