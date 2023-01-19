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

	repositoryArns := []string{}

	paginator := func(output *ecr.DescribeRepositoriesOutput, lastPage bool) bool {
		for _, repository := range output.Repositories {
			if shouldIncludeECRRepository(repository, excludeAfter, configObj) {
				repositoryArns = append(repositoryArns, aws.StringValue(repository.RepositoryArn))
			}
		}
		return !lastPage
	}

	param := &ecr.DescribeRepositoriesInput{}

	err := svc.DescribeRepositoriesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)

	}

	return repositoryArns, nil
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

func nukeAllECRRepositories(session *session.Session, arns []string) error {

	svc := ecr.New(session)

	if len(arns) == 0 {
		logging.Logger.Debugf("No ECR repositories to nuke in region %s", *session.Config.Region)
		return nil
	}

	var deletedArns []*string

	for _, arn := range arns {
		params := &ecr.DeleteRepositoryInput{
			Force:      aws.Bool(true),
			RegistryId: aws.String(arn),
		}

		_, err := svc.DeleteRepository(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   arn,
			ResourceType: "ECR Repository",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {

			deletedArns = append(deletedArns, aws.String(arn))
			logging.Logger.Debugf("Deleted ECR Repository: %s", arn)
		}
	}

	logging.Logger.Debugf("[OK] %d ECR Repositories deleted in %s", len(deletedArns), *session.Config.Region)

	return nil

}
