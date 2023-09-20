package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/aws/aws-sdk-go/service/securityhub/securityhubiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecurityHub struct {
	Client  securityhubiface.SecurityHubAPI
	Region  string
	HubArns []string
}

func (sh *SecurityHub) Init(session *session.Session) {
	sh.Client = securityhub.New(session)
}

func (sh *SecurityHub) ResourceName() string {
	return "security-hub"
}

func (sh *SecurityHub) ResourceIdentifiers() []string {
	return sh.HubArns
}

func (sh *SecurityHub) MaxBatchSize() int {
	return 5
}

func (sh *SecurityHub) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sh.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sh.HubArns = awsgo.StringValueSlice(identifiers)
	return sh.HubArns, nil
}

func (sh *SecurityHub) Nuke(identifiers []string) error {
	if err := sh.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
