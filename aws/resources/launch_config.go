package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Launch config Names
func (lc *LaunchConfigs) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := lc.Client.DescribeLaunchConfigurationsWithContext(lc.Context, &autoscaling.DescribeLaunchConfigurationsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var configNames []*string
	for _, c := range result.LaunchConfigurations {
		if configObj.LaunchConfiguration.ShouldInclude(config.ResourceValue{
			Time: c.CreatedTime,
			Name: c.LaunchConfigurationName,
		}) {
			configNames = append(configNames, c.LaunchConfigurationName)
		}
	}

	return configNames, nil
}

// Deletes all Launch configurations
func (lc *LaunchConfigs) nukeAll(configNames []*string) error {

	if len(configNames) == 0 {
		logging.Debugf("No Launch Configurations to nuke in region %s", lc.Region)
		return nil
	}

	logging.Debugf("Deleting all Launch Configurations in region %s", lc.Region)
	var deletedConfigNames []*string

	for _, configName := range configNames {
		params := &autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: configName,
		}

		_, err := lc.Client.DeleteLaunchConfigurationWithContext(lc.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(configName),
			ResourceType: "Launch configuration",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s", err)
		} else {
			deletedConfigNames = append(deletedConfigNames, configName)
			logging.Debugf("Deleted Launch configuration: %s", *configName)
		}
	}

	logging.Debugf("[OK] %d Launch Configuration(s) deleted in %s", len(deletedConfigNames), lc.Region)
	return nil
}
