package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllConfigRecorders(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := configservice.New(session)

	configRecorderNames := []string{}

	param := &configservice.DescribeConfigurationRecordersInput{}

	output, err := svc.DescribeConfigurationRecorders(param)
	if err != nil {
		return []string{}, errors.WithStackTrace(err)
	}

	for _, configRecorder := range output.ConfigurationRecorders {
		if shouldIncludeConfigRecorder(configRecorder, excludeAfter, configObj) {
			configRecorderNames = append(configRecorderNames, aws.StringValue(configRecorder.Name))
		}
	}

	return configRecorderNames, nil
}

func shouldIncludeConfigRecorder(configRecorder *configservice.ConfigurationRecorder, excludeAfter time.Time, configObj config.Config) bool {
	if configRecorder == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(configRecorder.Name),
		configObj.ConfigServiceRecorder.IncludeRule.NamesRegExp,
		configObj.ConfigServiceRecorder.ExcludeRule.NamesRegExp,
	)
}

func nukeAllConfigRecorders(session *session.Session, configRecorderNames []string) error {
	svc := configservice.New(session)

	if len(configRecorderNames) == 0 {
		logging.Logger.Debugf("No Config recorders to nuke in region %s", *session.Config.Region)
		return nil
	}

	var deletedNames []*string

	for _, configRecorderName := range configRecorderNames {
		params := &configservice.DeleteConfigurationRecorderInput{
			ConfigurationRecorderName: aws.String(configRecorderName),
		}

		_, err := svc.DeleteConfigurationRecorder(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   configRecorderName,
			ResourceType: "Config Recorder",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, aws.String(configRecorderName))
			logging.Logger.Debugf("Deleted Config Recorder: %s", configRecorderName)
		}
	}

	logging.Logger.Debugf("[OK] %d Config Recorders deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}
