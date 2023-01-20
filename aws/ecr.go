package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllECRRepositories(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := ecr.New(session)

	repositoryNames := []string{}

	paginator := func(output *ecr.DescribeRepositoriesOutput, lastPage bool) bool {
		for _, repository := range output.Repositories {
			if shouldIncludeECRRepository(repository, excludeAfter, configObj) {
				repositoryNames = append(repositoryNames, aws.StringValue(repository.RepositoryName))
			}
		}
		return !lastPage
	}

	param := &ecr.DescribeRepositoriesInput{}

	err := svc.DescribeRepositoriesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)

	}

	return repositoryNames, nil
}

func shouldIncludeECRRepository(repository *ecr.Repository, excludeAfter time.Time, configObj config.Config) bool {
	if repository == nil {
		return false
	}

	createdAtVal := aws.TimeValue(repository.CreatedAt)

	if excludeAfter.Before(createdAtVal) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(repository.RepositoryName),
		configObj.ECRRepository.IncludeRule.NamesRegExp,
		configObj.ECRRepository.ExcludeRule.NamesRegExp,
	)

}

func nukeAllECRRepositories(session *session.Session, repositoryNames []string) error {
	svc := ecr.New(session)

	if len(repositoryNames) == 0 {
		logging.Logger.Debugf("No ECR repositories to nuke in region %s", *session.Config.Region)
		return nil
	}

	var deletedNames []*string

	for _, repositoryName := range repositoryNames {
		params := &ecr.DeleteRepositoryInput{
			Force:          aws.Bool(true),
			RepositoryName: aws.String(repositoryName),
		}

		_, err := svc.DeleteRepository(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   repositoryName,
			ResourceType: "ECR Repository",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {

			deletedNames = append(deletedNames, aws.String(repositoryName))
			logging.Logger.Debugf("Deleted ECR Repository: %s", repositoryName)
		}
	}

	logging.Logger.Debugf("[OK] %d ECR Repositories deleted in %s", len(deletedNames), *session.Config.Region)

	return nil

}
