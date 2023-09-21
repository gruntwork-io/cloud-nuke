package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMGroups - represents all IAMGroups on the AWS Account
type IAMGroups struct {
	Client     iamiface.IAMAPI
	GroupNames []string
}

func (ig *IAMGroups) Init(session *session.Session) {
	ig.Client = iam.New(session)
}

// ResourceName - the simple name of the AWS resource
func (ig *IAMGroups) ResourceName() string {
	return "iam-group"
}

// ResourceIdentifiers - The IAM GroupNames
func (ig *IAMGroups) ResourceIdentifiers() []string {
	return ig.GroupNames
}

// Tentative batch size to ensure AWS doesn't throttle
// There's a global max of 500 groups so it shouldn't take long either way
func (ig *IAMGroups) MaxBatchSize() int {
	return 49
}

func (ig *IAMGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ig.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ig.GroupNames = awsgo.StringValueSlice(identifiers)
	return ig.GroupNames, nil
}

// Nuke - Destroy every group in this collection
func (ig *IAMGroups) Nuke(identifiers []string) error {
	if err := ig.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
