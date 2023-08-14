package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (csr *ConfigServiceRecorders) getAll(configObj config.Config) ([]*string, error) {

	configRecorderNames := []*string{}

	param := &configservice.DescribeConfigurationRecordersInput{}
	output, err := csr.Client.DescribeConfigurationRecorders(param)
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}

	for _, configRecorder := range output.ConfigurationRecorders {
		if configObj.ConfigServiceRecorder.ShouldInclude(config.ResourceValue{
			Name: configRecorder.Name,
		}) {
			configRecorderNames = append(configRecorderNames, configRecorder.Name)
		}
	}

	return configRecorderNames, nil
}

func (csr *ConfigServiceRecorders) nukeAll(configRecorderNames []string) error {
	if len(configRecorderNames) == 0 {
		logging.Logger.Debugf("No Config recorders to nuke in region %s", csr.Region)
		return nil
	}

	var deletedNames []*string

	for _, configRecorderName := range configRecorderNames {
		params := &configservice.DeleteConfigurationRecorderInput{
			ConfigurationRecorderName: aws.String(configRecorderName),
		}

		_, err := csr.Client.DeleteConfigurationRecorder(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   configRecorderName,
			ResourceType: "Config Recorder",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Config Recorder",
			}, map[string]interface{}{
				"region": csr.Region,
			})
		} else {
			deletedNames = append(deletedNames, aws.String(configRecorderName))
			logging.Logger.Debugf("Deleted Config Recorder: %s", configRecorderName)
		}
	}

	logging.Logger.Debugf("[OK] %d Config Recorders deleted in %s", len(deletedNames), csr.Region)
	return nil
}
