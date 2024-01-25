package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
)

// BaseAwsResource This BaseAwsResource struct and its associated methods to serve as a placeholder or template for a resource
// that is not yet fully implemented within a system or framework.
// Its purpose is to provide a skeleton structure that adheres to a specific interface or contract expected by the
// system without containing the actual implementation details.
type BaseAwsResource struct{}

func (umpl *BaseAwsResource) Init(_ *session.Session) {}
func (umpl *BaseAwsResource) ResourceName() string {
	return "not implemented: ResourceName"
}
func (umpl *BaseAwsResource) ResourceIdentifiers() []string {
	return nil
}
func (umpl *BaseAwsResource) MaxBatchSize() int {
	return 0
}
func (umpl *BaseAwsResource) Nuke(_ []string) error {
	return errors.New("not implemented: Nuke")
}
func (umpl *BaseAwsResource) GetAndSetIdentifiers(_ context.Context, _ config.Config) ([]string, error) {
	return nil, errors.New("not implemented: GetAndSetIdentifiers")
}
func (umpl *BaseAwsResource) IsNukable(_ string) (bool, error) {
	return false, errors.New("not implemented yet.")
}
