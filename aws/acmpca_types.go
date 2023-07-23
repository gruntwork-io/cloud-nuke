package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca/acmpcaiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// ACMPA - represents all ACMPA
type ACMPCA struct {
	Client acmpcaiface.ACMPCAAPI
	Region string
	ARNs   []string
}

// ResourceName - the simple name of the aws resource
func (ap ACMPCA) ResourceName() string {
	return "acmpca"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ap ACMPCA) ResourceIdentifiers() []string {
	return ap.ARNs
}

func (ap ACMPCA) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

// Nuke - nuke 'em all!!!
func (ap ACMPCA) Nuke(session *session.Session, arns []string) error {
	if err := ap.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
