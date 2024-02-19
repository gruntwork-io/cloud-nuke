package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// BaseAwsResource struct and its associated methods to serve as a placeholder or template for a resource that is not
// yet fully implemented within a system or framework. Its purpose is to provide a skeleton structure that adheres to a
// specific interface or contract expected by the system without containing the actual implementation details.
type BaseAwsResource struct {
	// A key-value of identifiers and nukable status
	Nukables map[string]error
}

func (umpl *BaseAwsResource) Init(_ *session.Session) {
	umpl.Nukables = make(map[string]error)
}
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

func (umpl *BaseAwsResource) GetNukableStatus(identifier string) (error, bool) {
	val, ok := umpl.Nukables[identifier]
	return val, ok
}

func (umpl *BaseAwsResource) SetNukableStatus(identifier string, err error) {
	umpl.Nukables[identifier] = err
}

// VerifyNukablePermissions performs nukable permission verification for each ID. For each ID, the function is
// executed, and the result (error or success) is recorded using the SetNukableStatus method, indicating whether
// the specified action is nukable
func (umpl *BaseAwsResource) VerifyNukablePermissions(ids []*string, nukableCheckfn func(id *string) error) {
	for _, id := range ids {
		// skip if the id is already exists
		if _, ok := umpl.GetNukableStatus(*id); ok {
			continue
		}
		err := nukableCheckfn(id)
		umpl.SetNukableStatus(*id, util.TransformAWSError(err))
	}
}

func (umpl *BaseAwsResource) IsNukable(identifier string) (bool, error) {
	err, ok := umpl.Nukables[identifier]
	if !ok {
		return false, fmt.Errorf("-")
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
