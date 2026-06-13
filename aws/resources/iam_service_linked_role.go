package resources

import (
	"context"
	goerr "errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMServiceLinkedRolesAPI defines the interface for IAM Service Linked Role operations.
type IAMServiceLinkedRolesAPI interface {
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	DeleteServiceLinkedRole(ctx context.Context, params *iam.DeleteServiceLinkedRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteServiceLinkedRoleOutput, error)
	GetServiceLinkedRoleDeletionStatus(ctx context.Context, params *iam.GetServiceLinkedRoleDeletionStatusInput, optFns ...func(*iam.Options)) (*iam.GetServiceLinkedRoleDeletionStatusOutput, error)
}

// NewIAMServiceLinkedRoles creates a new IAMServiceLinkedRoles resource using the generic resource pattern.
func NewIAMServiceLinkedRoles() AwsResource {
	return NewAwsResource(&resource.Resource[IAMServiceLinkedRolesAPI]{
		ResourceTypeName: "iam-service-linked-role",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[IAMServiceLinkedRolesAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.IAMServiceLinkedRoles
		},
		Lister: listIAMServiceLinkedRoles,
		// Use SequentialDeleter because each deletion requires async polling for status
		Nuker: resource.SequentialDeleter(deleteIAMServiceLinkedRole),
	})
}

// listIAMServiceLinkedRoles retrieves all IAM Service Linked Roles that match the config filters.
func listIAMServiceLinkedRoles(ctx context.Context, client IAMServiceLinkedRolesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allRoles []*string

	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, role := range page.Roles {
			// Only include service-linked roles (those with "aws-service-role" in ARN)
			if !strings.Contains(aws.ToString(role.Arn), "aws-service-role") {
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: role.CreateDate,
				Name: role.RoleName,
			}) {
				allRoles = append(allRoles, role.RoleName)
			}
		}
	}

	return allRoles, nil
}

// deleteIAMServiceLinkedRole deletes a single IAM Service Linked Role and waits for deletion to complete.
// Service Linked Role deletion is async - we must poll GetServiceLinkedRoleDeletionStatus until complete.
func deleteIAMServiceLinkedRole(ctx context.Context, client IAMServiceLinkedRolesAPI, roleName *string) error {
	// Initiate deletion - this returns a deletion task ID
	deletionData, err := client.DeleteServiceLinkedRole(ctx, &iam.DeleteServiceLinkedRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Poll for deletion completion
	return util.PollUntil(ctx, fmt.Sprintf("IAM ServiceLinked Role %s deletion", aws.ToString(roleName)), 3*time.Second, 5*time.Minute,
		func(ctx context.Context) (bool, error) {
			status, err := client.GetServiceLinkedRoleDeletionStatus(ctx, &iam.GetServiceLinkedRoleDeletionStatusInput{
				DeletionTaskId: deletionData.DeletionTaskId,
			})
			if err != nil {
				// Some roles delete so quickly on the AWS side that the deletion task
				// record is purged before we poll for its status, yielding a
				// NoSuchEntity (404) error. That alone does not confirm the role is
				// gone (the task ID could be invalid for other reasons), so verify the
				// role itself via GetRole before reporting success. This requires the
				// iam:GetRole permission in addition to the service-linked-role
				// deletion permissions.
				var notFoundErr *types.NoSuchEntityException
				if goerr.As(err, &notFoundErr) {
					_, roleErr := client.GetRole(ctx, &iam.GetRoleInput{
						RoleName: roleName,
					})
					var roleNotFound *types.NoSuchEntityException
					switch {
					case goerr.As(roleErr, &roleNotFound):
						// Role is gone: deletion succeeded and AWS purged the task record.
						return true, nil
					case roleErr != nil:
						// Could not confirm the role's state; surface the verification
						// error rather than masking it with the task-not-found error.
						return false, errors.WithStackTrace(roleErr)
					default:
						// The task record is gone but GetRole still sees the role. AWS
						// already accepted the deletion, so this is almost always IAM
						// read-after-write lag rather than a real failure. Keep polling
						// (as we do for an in-progress deletion): the role should
						// disappear on a later check, and the surrounding timeout still
						// bounds a genuinely stuck delete.
						logging.Debugf("Deletion task for IAM ServiceLinked Role %s is gone but the role is still visible; retrying", aws.ToString(roleName))
						return false, nil
					}
				}
				return false, errors.WithStackTrace(err)
			}

			switch status.Status {
			case types.DeletionTaskStatusTypeSucceeded:
				return true, nil
			case types.DeletionTaskStatusTypeInProgress:
				logging.Debugf("Deletion of IAM ServiceLinked Role %s is still in progress", aws.ToString(roleName))
				return false, nil
			default:
				return false, fmt.Errorf("failed with status %s", string(status.Status))
			}
		})
}
