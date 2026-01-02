package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMServiceLinkedRolesAPI defines the interface for IAM Service Linked Role operations.
type IAMServiceLinkedRolesAPI interface {
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	DeleteServiceLinkedRole(ctx context.Context, params *iam.DeleteServiceLinkedRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteServiceLinkedRoleOutput, error)
	GetServiceLinkedRoleDeletionStatus(ctx context.Context, params *iam.GetServiceLinkedRoleDeletionStatusInput, optFns ...func(*iam.Options)) (*iam.GetServiceLinkedRoleDeletionStatusOutput, error)
}

// NewIAMServiceLinkedRoles creates a new IAMServiceLinkedRoles resource using the generic resource pattern.
func NewIAMServiceLinkedRoles() AwsResource {
	return NewAwsResource(&resource.Resource[IAMServiceLinkedRolesAPI]{
		ResourceTypeName: "iam-service-linked-role",
		BatchSize:        49,
		IsGlobal:         true,
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

	// Wait for the deletion to start
	time.Sleep(3 * time.Second)

	// Poll for deletion completion
	for {
		status, err := client.GetServiceLinkedRoleDeletionStatus(ctx, &iam.GetServiceLinkedRoleDeletionStatusInput{
			DeletionTaskId: deletionData.DeletionTaskId,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}

		switch status.Status {
		case types.DeletionTaskStatusTypeSucceeded:
			return nil
		case types.DeletionTaskStatusTypeInProgress:
			logging.Debugf("Deletion of IAM ServiceLinked Role %s is still in progress", aws.ToString(roleName))
			time.Sleep(3 * time.Second)
		default:
			// Failed or unknown status
			return errors.WithStackTrace(fmt.Errorf("deletion of IAM ServiceLinked Role %s failed with status %s", aws.ToString(roleName), string(status.Status)))
		}
	}
}
