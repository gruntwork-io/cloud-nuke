package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// Returns a formatted string of ConfigRules
func getAllConfigRecorders(session *session.Session, region string) ([]*string, error) {
	svc := configservice.New(session)
	result, err := svc.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, recorder := range result.ConfigurationRecorders {
		names = append(names, recorder.Name)
	}

	return names, nil
}

// Returns a formatted string of ConfigRules
func getAllConfigRules(session *session.Session, region string) ([]*string, error) {
	svc := configservice.New(session)
	result, err := svc.DescribeConfigRules(&configservice.DescribeConfigRulesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ruleNames []*string
	for _, rule := range result.ConfigRules {
		ruleNames = append(ruleNames, rule.ConfigRuleName)
	}

	return ruleNames, nil
}

func deleteConfigRule(svc *configservice.ConfigService, ruleName *string) error {
	_, err := svc.DeleteConfigRule(&configservice.DeleteConfigRuleInput{
		ConfigRuleName: ruleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Delete all GuardDuty Detectors
func nukeAllConfigRules(session *session.Session, ruleNames []*string) error {
	if len(ruleNames) == 0 {
		logging.Logger.Info("No Config Rules to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all Config Rules")

	deletedUsers := 0
	svc := configservice.New(session)
	multiErr := new(multierror.Error)

	for _, name := range ruleNames {
		err := deleteConfigRule(svc, name)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedUsers++
			logging.Logger.Infof("Deleted Config Rule: %s", *name)
		}
	}

	logging.Logger.Infof("[OK] %d Config Rule(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}

func deleteConfigRecorder(svc *configservice.ConfigService, recorderName *string) error {
	_, err := svc.DeleteConfigurationRecorder(&configservice.DeleteConfigurationRecorderInput{
		ConfigurationRecorderName: recorderName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Delete all Config Service Recorders
func nukeAllConfigRecorders(session *session.Session, recorders []*string) error {
	if len(recorders) == 0 {
		logging.Logger.Info("No Config Recorders to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all Config Recorders")

	deletedUsers := 0
	svc := configservice.New(session)
	multiErr := new(multierror.Error)

	for _, name := range recorders {
		err := deleteConfigRecorder(svc, name)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedUsers++
			logging.Logger.Infof("Deleted Config Recorder: %s", *name)
		}
	}

	logging.Logger.Infof("[OK] %d Config Recorder(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}
