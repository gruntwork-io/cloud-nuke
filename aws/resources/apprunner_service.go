package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (a *AppRunnerService) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[App Runner Service] No App Runner Services found in region %s", a.Region)
		return nil
	}

	logging.Debugf("[App Runner Service] Deleting all App Runner Services in region %s", a.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		logging.Debugf("[App Runner Service] Deleting App Runner Service %s in region %s", *identifier, a.Region)

		_, err := a.Client.DeleteService(a.Context, &apprunner.DeleteServiceInput{
			ServiceArn: identifier,
		})
		if err != nil {
			logging.Debugf("[App Runner Service] Error deleting App Runner Service %s in region %s", *identifier, a.Region)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[App Runner Service] Deleted App Runner Service %s in region %s", *identifier, a.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: a.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d App Runner Service(s) nuked in %s", len(deleted), a.Region)
	return nil
}

func (a *AppRunnerService) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	paginator := apprunner.NewListServicesPaginator(a.Client, &apprunner.ListServicesInput{
		MaxResults: aws.Int32(19),
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(c)
		if err != nil {
			logging.Debugf("[App Runner Service] Failed to list app runner services: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, service := range output.ServiceSummaryList {
			if configObj.AppRunnerService.ShouldInclude(config.ResourceValue{
				Name: service.ServiceName,
				Time: service.CreatedAt,
			}) {
				identifiers = append(identifiers, service.ServiceArn)
			}
		}
	}

	return identifiers, nil
}
