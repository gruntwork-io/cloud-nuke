package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
)

type AwsAccountResources struct {
	Resources map[string]AwsRegionResource
}

type AwsResources interface {
	ResourceName() string
	ResourceIdentifiers() []string
	Nuke(session *session.Session) error
	NukeBatch(session *session.Session, identifiers []string) error
}

type AwsRegionResource struct {
	Resources []AwsResources
}
