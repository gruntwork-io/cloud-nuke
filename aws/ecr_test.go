package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
	defer deleteECRRepository(t, region, repositoryName, true)

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

	defer deleteECRRepository(t, region, repositoryName, true)

	identifiers := []*string{repositoryName}

	require.NoError(
		t,
		nukeAllECRRepositories(session, aws.StringValueSlice(identifiers)),
	)

	assertECRRepositoriesDeleted(t, region, identifiers)
}

func assertECRRepositoriesDeleted(t *testing.T, region string, repositoryNames []*string) {

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := ecr.New(session)

	// Try to figure out which registries we have in play
	registryParam := &ecr.DescribeRegistryInput{}

	registryResp, registryErr := svc.DescribeRegistry(registryParam)
	require.NoError(t, registryErr)

	registryID := registryResp.RegistryId

	param := &ecr.DescribeRepositoriesInput{
		RegistryId:      registryID,
		RepositoryNames: repositoryNames,
	}

	resp, err := svc.DescribeRepositories(param)

	require.NoError(t, err)
	if len(resp.Repositories) > 0 {
		t.Logf("Repository: %+v\n", resp.Repositories)
		t.Fatalf("At least one of the following ECR Repositories was not deleted: %+v\n", aws.StringValueSlice(repositoryNames))
	}
}
