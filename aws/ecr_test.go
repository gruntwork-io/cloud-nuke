package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListECRRepositories(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	repositoryName := createECRRepository(t, region)
	defer deleteECRRepository(t, region, repositoryName, false)

	time.Sleep(10 * time.Second)

	repositoryNames, err := getAllECRRepositories(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, repositoryNames, aws.StringValue(repositoryName))
}

func deleteECRRepository(t *testing.T, region string, repositoryName *string, checkErr bool) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	ecrService := ecr.New(session)

	param := &ecr.DeleteRepositoryInput{
		RepositoryName: repositoryName,
	}

	_, deleteErr := ecrService.DeleteRepository(param)
	if checkErr {
		require.NoError(t, deleteErr)
	}
}

func createECRRepository(t *testing.T, region string) *string {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	ecrService := ecr.New(session)

	name := strings.ToLower(fmt.Sprintf("cloud-nuke-test-%s-%s", util.UniqueID(), util.UniqueID()))

	param := &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(name),
	}

	output, createRepositoryErr := ecrService.CreateRepository(param)

	require.NoError(t, createRepositoryErr)

	return output.Repository.RepositoryName

}

func TestNukeECRRepositoryOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	repositoryName := createECRRepository(t, region)
	defer deleteECRRepository(t, region, repositoryName, false)

	identifiers := []*string{repositoryName}

	require.NoError(
		t,
		nukeAllECRRepositories(session, aws.StringValueSlice(identifiers)),
	)

	assertECRRepositoriesDeleted(t, region, identifiers)
}

func TestNukeECRRepositoryMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	repositoryNames := []*string{}
	for i := 0; i < 3; i++ {
		repositoryName := createECRRepository(t, region)
		defer deleteECRRepository(t, region, repositoryName, false)
		repositoryNames = append(repositoryNames, repositoryName)
	}

	require.NoError(
		t,
		nukeAllECRRepositories(session, aws.StringValueSlice(repositoryNames)),
	)

	assertECRRepositoriesDeleted(t, region, repositoryNames)
}

func assertECRRepositoriesDeleted(t *testing.T, region string, repositoryNames []*string) {

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := ecr.New(session)

	param := &ecr.DescribeRepositoriesInput{
		RepositoryNames: repositoryNames,
	}

	resp, err := svc.DescribeRepositories(param)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// If the repository can't be looked up because it doesn't exist...then our test has been successful
			case ecr.ErrCodeRepositoryNotFoundException:
				t.Log("Ignoring repository not found error in test lookup")
			default:
				require.NoError(t, err)
				if len(resp.Repositories) > 0 {
					t.Logf("Repository: %+v\n", resp.Repositories)
					t.Fatalf("At least one of the following ECR Repositories was not deleted: %+v\n", aws.StringValueSlice(repositoryNames))
				}
			}
		}
	}
}
