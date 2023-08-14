package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (registry *ECR) getAll(configObj config.Config) ([]*string, error) {
	repositoryNames := []*string{}

	paginator := func(output *ecr.DescribeRepositoriesOutput, lastPage bool) bool {
		for _, repository := range output.Repositories {
			if configObj.ECRRepository.ShouldInclude(config.ResourceValue{
				Time: repository.CreatedAt,
				Name: repository.RepositoryName,
			}) {
				repositoryNames = append(repositoryNames, repository.RepositoryName)
			}
		}
		return !lastPage
	}

	param := &ecr.DescribeRepositoriesInput{}
	err := registry.Client.DescribeRepositoriesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return repositoryNames, nil
}

func (registry *ECR) nukeAll(repositoryNames []string) error {
	if len(repositoryNames) == 0 {
		logging.Logger.Debugf("No ECR repositories to nuke in region %s", registry.Region)
		return nil
	}

	var deletedNames []*string

	for _, repositoryName := range repositoryNames {
		params := &ecr.DeleteRepositoryInput{
			Force:          aws.Bool(true),
			RepositoryName: aws.String(repositoryName),
		}

		_, err := registry.Client.DeleteRepository(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   repositoryName,
			ResourceType: "ECR Repository",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECR Repo",
			}, map[string]interface{}{
				"region": registry.Region,
			})
			logging.Logger.Debugf("[Failed] %s", err)
		} else {

			deletedNames = append(deletedNames, aws.String(repositoryName))
			logging.Logger.Debugf("Deleted ECR Repository: %s", repositoryName)
		}
	}

	logging.Logger.Debugf("[OK] %d ECR Repositories deleted in %s", len(deletedNames), registry.Region)

	return nil
}
