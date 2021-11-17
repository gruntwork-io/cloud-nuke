package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllCloudWatchLogGroups(session *session.Session, region string) ([]*string, error) {
	svc := cloudwatchlogs.New(session)

	output, err := svc.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, loggroup := range output.LogGroups {
		names = append(names, loggroup.LogGroupName)
	}

	return names, nil
}

func nukeAllCloudWatchLogGroups(session *session.Session, identifiers []*string) error {
	svc := cloudwatchlogs.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No CloudWatch Log Groups to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all CloudWatch Log Groups in region %s", *session.Config.Region)

	var deleteResources = 0

	for _, name := range identifiers {
		_, err := svc.DeleteLogGroup(&cloudwatchlogs.DeleteLogGroupInput{
			LogGroupName: name,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			logging.Logger.Infof("[OK] CloudWatch Log Group %s terminated in %s", *name, *session.Config.Region)
			deleteResources++
		}

	}

	logging.Logger.Infof("[OK] %d CloudWatch Log Group(s) terminated in %s", deleteResources, *session.Config.Region)

	return nil
}
