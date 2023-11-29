package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/acmpca/acmpcaiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// ACMPA - represents all ACMPA
type ACMPCA struct {
	Client acmpcaiface.ACMPCAAPI
	Region string
	ARNs   []string
}

func (ap *ACMPCA) Init(session *session.Session) {
	ap.Client = acmpca.New(session)
}

// ResourceName - the simple name of the aws resource
func (ap *ACMPCA) ResourceName() string {
	return "acmpca"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ap *ACMPCA) ResourceIdentifiers() []string {
	return ap.ARNs
}

func (ap *ACMPCA) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

func (ap *ACMPCA) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ap.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ap.ARNs = awsgo.StringValueSlice(identifiers)
	return ap.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (ap *ACMPCA) Nuke(arns []string) error {
	if err := ap.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
