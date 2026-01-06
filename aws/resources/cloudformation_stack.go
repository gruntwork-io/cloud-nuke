package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudFormationStacksAPI defines the interface for CloudFormation stack operations.
type CloudFormationStacksAPI interface {
	ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

// activeStackStatuses defines the stack statuses we care about (excluding DELETE_COMPLETE, DELETE_IN_PROGRESS).
// These are the statuses for stacks that exist and can potentially be deleted.
var activeStackStatuses = []types.StackStatus{
	types.StackStatusCreateInProgress,
	types.StackStatusCreateFailed,
	types.StackStatusCreateComplete,
	types.StackStatusRollbackInProgress,
	types.StackStatusRollbackFailed,
	types.StackStatusRollbackComplete,
	types.StackStatusDeleteFailed,
	types.StackStatusUpdateInProgress,
	types.StackStatusUpdateCompleteCleanupInProgress,
	types.StackStatusUpdateComplete,
	types.StackStatusUpdateFailed,
	types.StackStatusUpdateRollbackInProgress,
	types.StackStatusUpdateRollbackFailed,
	types.StackStatusUpdateRollbackCompleteCleanupInProgress,
	types.StackStatusUpdateRollbackComplete,
	types.StackStatusReviewInProgress,
	types.StackStatusImportInProgress,
	types.StackStatusImportComplete,
	types.StackStatusImportRollbackInProgress,
	types.StackStatusImportRollbackFailed,
	types.StackStatusImportRollbackComplete,
}

// NewCloudFormationStacks creates a new CloudFormationStacks resource using the generic resource pattern.
func NewCloudFormationStacks() AwsResource {
	return NewAwsResource(&resource.Resource[CloudFormationStacksAPI]{
		ResourceTypeName: "cloudformation-stack",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudFormationStacksAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = cloudformation.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudFormationStack
		},
		Lister: listCloudFormationStacks,
		Nuker:  resource.SimpleBatchDeleter(deleteCloudFormationStack),
	})
}

// listCloudFormationStacks retrieves all CloudFormation stacks that match the config filters.
func listCloudFormationStacks(ctx context.Context, client CloudFormationStacksAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var stackNames []*string

	paginator := cloudformation.NewListStacksPaginator(client, &cloudformation.ListStacksInput{
		StackStatusFilter: activeStackStatuses,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, stack := range page.StackSummaries {
			if shouldIncludeStack(ctx, client, &stack, cfg) {
				stackNames = append(stackNames, stack.StackName)
			}
		}
	}

	return stackNames, nil
}

// shouldIncludeStack determines if a CloudFormation stack should be included for deletion.
func shouldIncludeStack(ctx context.Context, client CloudFormationStacksAPI, stack *types.StackSummary, cfg config.ResourceType) bool {
	if stack == nil {
		return false
	}

	// Get detailed stack information including tags
	stackDetails, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: stack.StackName,
	})
	if err != nil {
		logging.Debugf("Failed to describe stack %s: %v", aws.ToString(stack.StackName), err)
		return false
	}

	if len(stackDetails.Stacks) == 0 {
		return false
	}

	tags := util.ConvertCloudFormationTagsToMap(stackDetails.Stacks[0].Tags)

	return cfg.ShouldInclude(config.ResourceValue{
		Name: stack.StackName,
		Time: stack.CreationTime,
		Tags: tags,
	})
}

// deleteCloudFormationStack deletes a single CloudFormation stack.
func deleteCloudFormationStack(ctx context.Context, client CloudFormationStacksAPI, stackName *string) error {
	_, err := client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: stackName,
	})
	return err
}
