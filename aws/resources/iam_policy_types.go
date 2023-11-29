package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMPolicies - represents all IAM Policies on the AWS account
type IAMPolicies struct {
	Client     iamiface.IAMAPI
	PolicyArns []string
}

func (ip *IAMPolicies) Init(session *session.Session) {
	ip.Client = iam.New(session)
}

// ResourceName - the simple name of the AWS resource
func (ip *IAMPolicies) ResourceName() string {
	return "iam-policy"
}

// ResourceIdentifiers - The IAM GroupNames
func (ip *IAMPolicies) ResourceIdentifiers() []string {
	return ip.PolicyArns
}

// MaxBatchSize Tentative batch size to ensure AWS doesn't throttle
func (ip *IAMPolicies) MaxBatchSize() int {
	return 20
}

func (ip *IAMPolicies) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ip.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ip.PolicyArns = awsgo.StringValueSlice(identifiers)
	return ip.PolicyArns, nil
}

// Nuke - Destroy every group in this collection
func (ip *IAMPolicies) Nuke(identifiers []string) error {
	if err := ip.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
