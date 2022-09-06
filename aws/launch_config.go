package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Launch config Names
func getAllLaunchConfigurations(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := autoscaling.New(session)
	result, err := svc.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var configNames []*string
	for _, config := range result.LaunchConfigurations {
		if shouldIncludeLaunchConfiguration(config, excludeAfter, configObj) {
			configNames = append(configNames, config.LaunchConfigurationName)
		}
	}

	return configNames, nil
}

func shouldIncludeLaunchConfiguration(lc *autoscaling.LaunchConfiguration, excludeAfter time.Time, configObj config.Config) bool {
	if lc == nil {
		return false
	}

	if lc.CreatedTime != nil && excludeAfter.Before(*lc.CreatedTime) {
		return false
	}

	return config.ShouldInclude(
		awsgo.StringValue(lc.LaunchConfigurationName),
		configObj.LaunchConfiguration.IncludeRule.NamesRegExp,
		configObj.LaunchConfiguration.ExcludeRule.NamesRegExp,
	)
}

// Deletes all Launch configurations
func nukeAllLaunchConfigurations(session *session.Session, configNames []*string) error {
	svc := autoscaling.New(session)

	if len(configNames) == 0 {
		logging.Logger.Debugf("No Launch Configurations to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Launch Configurations in region %s", *session.Config.Region)
	var deletedConfigNames []*string

	for _, configName := range configNames {
		params := &autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: configName,
		}

		_, err := svc.DeleteLaunchConfiguration(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(configName),
			ResourceType: "Launch configuration",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedConfigNames = append(deletedConfigNames, configName)
			logging.Logger.Debugf("Deleted Launch configuration: %s", *configName)
		}
	}

	logging.Logger.Debugf("[OK] %d Launch Configuration(s) deleted in %s", len(deletedConfigNames), *session.Config.Region)
	return nil
}
