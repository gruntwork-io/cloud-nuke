package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
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

func (a *ACM) Init(session *session.Session) {
	a.Client = acm.New(session)
}

// ResourceName - the simple name of the aws resource
func (a *ACM) ResourceName() string {
	return "acm"
}

// ResourceIdentifiers - the arns of the aws certificate manager certificates
func (a *ACM) ResourceIdentifiers() []string {
	return a.ARNs
}

func (a *ACM) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

func (a *ACM) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := a.getAll(configObj)
	if err != nil {
		return nil, err
	}

	a.ARNs = awsgo.StringValueSlice(identifiers)
	return a.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (a *ACM) Nuke(arns []string) error {
	if err := a.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
