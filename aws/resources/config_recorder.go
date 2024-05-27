package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (csr *ConfigServiceRecorders) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	configRecorderNames := []*string{}

	param := &configservice.DescribeConfigurationRecordersInput{}
	output, err := csr.Client.DescribeConfigurationRecordersWithContext(csr.Context, param)
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
		logging.Debugf("No Config recorders to nuke in region %s", csr.Region)
		return nil
	}

	var deletedNames []*string

	for _, configRecorderName := range configRecorderNames {
		params := &configservice.DeleteConfigurationRecorderInput{
			ConfigurationRecorderName: aws.String(configRecorderName),
		}

		_, err := csr.Client.DeleteConfigurationRecorderWithContext(csr.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   configRecorderName,
			ResourceType: "Config Recorder",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, aws.String(configRecorderName))
			logging.Debugf("Deleted Config Recorder: %s", configRecorderName)
		}
	}

	logging.Debugf("[OK] %d Config Recorders deleted in %s", len(deletedNames), csr.Region)
	return nil
}
