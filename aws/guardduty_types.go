package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// GuardDutyDetectors - represents all GuardDutyDetectors on the AWS Account
type GuardDutyDetectors struct {
	Detectors []string
}

// ResourceName - the simple name of the aws resource
func (u GuardDutyDetectors) ResourceName() string {
	return "guardduty"
}

// ResourceIdentifiers - The IAM UserNames
func (u GuardDutyDetectors) ResourceIdentifiers() []string {
	return u.Detectors
}

// Tentative batch size to ensure AWS doesn't throttle
func (u GuardDutyDetectors) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (u GuardDutyDetectors) Nuke(session *session.Session, detectors []string) error {
	if err := nukeAllGuardDutyDetectors(session, awsgo.StringSlice(detectors)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
