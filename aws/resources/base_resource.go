package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
)

const maxRetries = 3
const maxStopRetries = 3
const waitDuration = 5 * time.Second
const stopWaitDuration = 5 * time.Second

// BaseAwsResource struct and its associated methods to serve as a placeholder or template for a resource that is not
// yet fully implemented within a system or framework. Its purpose is to provide a skeleton structure that adheres to a
// specific interface or contract expected by the system without containing the actual implementation details.
type BaseAwsResource struct {
	// A key-value of identifiers and nukable status
	Nukables map[string]error
	Timeout  time.Duration
	Context  context.Context
	cancel   context.CancelFunc
}

func (br *BaseAwsResource) InitV2(cfg aws.Config) {
	br.Nukables = make(map[string]error)
}

func (br *BaseAwsResource) ResourceName() string {
	return "not implemented: ResourceName"
}
func (br *BaseAwsResource) ResourceIdentifiers() []string {
	return nil
}
func (br *BaseAwsResource) MaxBatchSize() int {
	return 0
}
func (br *BaseAwsResource) Nuke(_ []string) error {
	return errors.New("not implemented: Nuke")
}
func (br *BaseAwsResource) GetAndSetIdentifiers(_ context.Context, _ config.Config) ([]string, error) {
	return nil, errors.New("not implemented: GetAndSetIdentifiers")
}

func (br *BaseAwsResource) GetNukableStatus(identifier string) (error, bool) {
	val, ok := br.Nukables[identifier]
	return val, ok
}

func (br *BaseAwsResource) SetNukableStatus(identifier string, err error) {
	br.Nukables[identifier] = err
}
func (br *BaseAwsResource) GetAndSetResourceConfig(_ config.Config) config.ResourceType {
	return config.ResourceType{
		Timeout: "",
	}
}

func (br *BaseAwsResource) PrepareContext(parentContext context.Context, resourceConfig config.ResourceType) error {
	if resourceConfig.Timeout == "" {
		br.Context = parentContext
		return nil
	}

	duration, err := time.ParseDuration(resourceConfig.Timeout)
	if err != nil {
		return err
	}

	br.Context, _ = context.WithTimeout(parentContext, duration)
	return nil
}

// VerifyNukablePermissions performs nukable permission verification for each ID. For each ID, the function is
// executed, and the result (error or success) is recorded using the SetNukableStatus method, indicating whether
// the specified action is nukable
func (br *BaseAwsResource) VerifyNukablePermissions(ids []*string, nukableCheckfn func(id *string) error) {
	// check if the 'Nukables' map is initialized, and if it's not, initialize it
	if br.Nukables == nil {
		br.Nukables = make(map[string]error)
	}

	for _, id := range ids {
		// skip if the id is already exists
		if _, ok := br.GetNukableStatus(*id); ok {
			continue
		}
		err := nukableCheckfn(id)
		br.SetNukableStatus(*id, util.TransformAWSError(err))
	}
}

func (br *BaseAwsResource) IsNukable(identifier string) (bool, error) {
	err, ok := br.Nukables[identifier]
	if !ok {
		return false, fmt.Errorf("-")
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (br *BaseAwsResource) IsUsingV2() bool {
	return true
}
