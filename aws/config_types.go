package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// ConfigService - represents all ConfigService on the AWS Account
type ConfigService struct {
	Rules []string
}

// ResourceName - the simple name of the aws resource
func (u ConfigService) ResourceName() string {
	return "config-rules"
}

// ResourceIdentifiers - The IAM UserNames
func (u ConfigService) ResourceIdentifiers() []string {
	return u.Rules
}

// Tentative batch size to ensure AWS doesn't throttle
func (u ConfigService) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (u ConfigService) Nuke(session *session.Session, detectors []string) error {
	if err := nukeAllConfigRules(session, awsgo.StringSlice(detectors)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// ConfigServiceRecorders - represents all ConfigServiceRecorders on the AWS Account
type ConfigServiceRecorders struct {
	Rules []string
}

// ResourceName - the simple name of the aws resource
func (u ConfigServiceRecorders) ResourceName() string {
	return "config-recorders"
}

// ResourceIdentifiers - The IAM UserNames
func (u ConfigServiceRecorders) ResourceIdentifiers() []string {
	return u.Rules
}

// Tentative batch size to ensure AWS doesn't throttle
func (u ConfigServiceRecorders) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (u ConfigServiceRecorders) Nuke(session *session.Session, detectors []string) error {
	if err := nukeAllConfigRecorders(session, awsgo.StringSlice(detectors)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
