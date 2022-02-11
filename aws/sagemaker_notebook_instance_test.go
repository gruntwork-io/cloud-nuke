package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// There's a built-in function WaitUntilDBInstanceAvailable but
// the times that it was tested, it wasn't returning anything so we'll leave with the
// custom one.

func waitUntilNotebookInstanceCreated(svc *sagemaker.SageMaker, name *string) error {
	input := &sagemaker.DescribeNotebookInstanceInput{
		NotebookInstanceName: name,
	}

	for i := 0; i < 600; i++ {
		instance, err := svc.DescribeNotebookInstance(input)
		status := instance.NotebookInstanceStatus

		if awsgo.StringValue(status) != "Pending" {
			return nil
		}

		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for SageMaker Notebook Instance to be created")
	}

	return SageMakerNotebookInstanceDeleteError{name: *name}
}

func createTestNotebookInstance(t *testing.T, session *session.Session, name string, roleArn string) {
	svc := sagemaker.New(session)

	params := &sagemaker.CreateNotebookInstanceInput{
		InstanceType:         awsgo.String("ml.t2.medium"),
		NotebookInstanceName: awsgo.String(name),
		RoleArn:              awsgo.String(roleArn),
	}

	_, err := svc.CreateNotebookInstance(params)
	require.NoError(t, err)

	waitUntilNotebookInstanceCreated(svc, &name)
}

func TestNukeNotebookInstance(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Config: awsgo.Config{
				Region: awsgo.String(region),
			},
		},
	)

	notebookName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	role := createNotebookRole(t, session, notebookName+"-role")
	defer deleteNotebookRole(session, role)

	createTestNotebookInstance(t, session, notebookName, *role.Arn)

	defer func() {
		nukeAllNotebookInstances(session, []*string{&notebookName})

		notebookNames, _ := getAllNotebookInstances(session, excludeAfter)

		assert.NotContains(t, awsgo.StringValueSlice(notebookNames), strings.ToLower(notebookName))
	}()

	instances, err := getAllNotebookInstances(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of SageMaker Notebook Instances", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(instances), notebookName)

}
