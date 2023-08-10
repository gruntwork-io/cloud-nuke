package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// ACMPA - represents all ACMPA
type ACM struct {
	Client acmiface.ACMAPI
	Region string
	ARNs   []string
}

func (acm *ACM) Init(session session.Session) {

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

func (acm ACM) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := acm.getAll(configObj)
	if err != nil {
		return nil, err
	}

	acm.ARNs = awsgo.StringValueSlice(identifiers)
	return acm.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (acm ACM) Nuke(arns []string) error {
	if err := acm.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
