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

// IAMUsers - represents all IAMUsers on the AWS Account
type IAMUsers struct {
	Client    iamiface.IAMAPI
	UserNames []string
}

func (iu *IAMUsers) Init(session *session.Session) {
	iu.Client = iam.New(session)
}

// ResourceName - the simple name of the aws resource
func (iu *IAMUsers) ResourceName() string {
	return "iam"
}

// ResourceIdentifiers - The IAM UserNames
func (iu *IAMUsers) ResourceIdentifiers() []string {
	return iu.UserNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (iu *IAMUsers) MaxBatchSize() int {
	return 49
}

func (iu *IAMUsers) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := iu.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	iu.UserNames = awsgo.StringValueSlice(identifiers)
	return iu.UserNames, nil
}

// Nuke - nuke 'em all!!!
func (iu *IAMUsers) Nuke(users []string) error {
	if err := iu.nukeAll(awsgo.StringSlice(users)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
