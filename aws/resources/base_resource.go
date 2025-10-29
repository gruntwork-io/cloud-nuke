package resources

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

const maxRetries = 3
const maxStopRetries = 3
const waitDuration = 5 * time.Second
const stopWaitDuration = 5 * time.Second

// BaseAwsResource embeds BaseCloudResource and adds AWS-specific functionality
// It serves as a base for all AWS resource implementations
type BaseAwsResource struct {
	resource.BaseCloudResource // Embedded cloud-agnostic base
}

func (br *BaseAwsResource) Init(cfg aws.Config) {
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

func (br *BaseAwsResource) GetAndSetResourceConfig(_ config.Config) config.ResourceType {
	return config.ResourceType{
		Timeout: "",
	}
}

// VerifyNukablePermissions performs nukable permission verification for each ID. For each ID, the function is
// executed, and the result (error or success) is recorded using the SetNukableStatus method, indicating whether
// the specified action is nukable (AWS-specific functionality)
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
