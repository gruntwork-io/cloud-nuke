package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// ACMPA - represents all ACMPA
type ACM struct {
	ARNs []string
}

// ResourceName - the simple name of the aws resource
func (acm ACM) ResourceName() string {
	return "acm"
}

// ResourceIdentifiers - the arns of the aws certificate manager certificates
func (acm ACM) ResourceIdentifiers() []string {
	return acm.ARNs
}

func (acm ACM) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

// Nuke - nuke 'em all!!!
func (acm ACM) Nuke(session *session.Session, arns []string) error {
	if err := nukeAllACMs(session, awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
