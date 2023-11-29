package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// List all IAM Roles in the AWS account
func (islr *IAMServiceLinkedRoles) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	allIAMServiceLinkedRoles := []*string{}
	err := islr.Client.ListRolesPages(
		&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, iamServiceLinkedRole := range page.Roles {
				if islr.shouldInclude(iamServiceLinkedRole, configObj) {
					allIAMServiceLinkedRoles = append(allIAMServiceLinkedRoles, iamServiceLinkedRole.RoleName)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, gruntworkerrors.WithStackTrace(err)
	}
	return allIAMServiceLinkedRoles, nil
}

func (islr *IAMServiceLinkedRoles) deleteIamServiceLinkedRole(roleName *string) error {
	// Deletion ID looks like this: "
	//{
	//	DeletionTaskId: "task/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling_2/d3c4c9fc-7fd3-4a36-974a-afb0eb78f102"
	//}
	deletionData, err := islr.Client.DeleteServiceLinkedRole(&iam.DeleteServiceLinkedRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}

	// Wait for the deletion to complete
	time.Sleep(3 * time.Second)

	var deletionStatus *iam.GetServiceLinkedRoleDeletionStatusOutput

	done := false
	for !done {
		done = true
		// Check if the deletion is complete
		deletionStatus, err = islr.Client.GetServiceLinkedRoleDeletionStatus(&iam.GetServiceLinkedRoleDeletionStatusInput{
			DeletionTaskId: deletionData.DeletionTaskId,
		})
		if err != nil {
			return gruntworkerrors.WithStackTrace(err)
		}
		if aws.StringValue(deletionStatus.Status) == "IN_PROGRESS" {
			logging.Debugf("Deletion of IAM ServiceLinked Role %s is still in progress", aws.StringValue(roleName))
			done = false
			time.Sleep(3 * time.Second)
		}

	}

	if aws.StringValue(deletionStatus.Status) != "SUCCEEDED" {
		err := fmt.Sprintf("Deletion of IAM ServiceLinked Role %s failed with status %s", aws.StringValue(roleName), aws.StringValue(deletionStatus.Status))
		return gruntworkerrors.WithStackTrace(errors.New(err))
	}

	return nil
}

// Delete all IAM Roles
func (islr *IAMServiceLinkedRoles) nukeAll(roleNames []*string) error {
	if len(roleNames) == 0 {
		logging.Debug("No IAM Service Linked Roles to nuke")
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on IAMRoles.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(roleNames) > 100 {
		logging.Debugf("Nuking too many IAM Service Linked Roles at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyIamRoleErr{}
	}

	// There is no bulk delete IAM Roles API, so we delete the batch of IAM roles concurrently using go routines
	logging.Debugf("Deleting all IAM Service Linked Roles")
	wg := new(sync.WaitGroup)
	wg.Add(len(roleNames))
	errChans := make([]chan error, len(roleNames))
	for i, roleName := range roleNames {
		errChans[i] = make(chan error, 1)
		go islr.deleteIamServiceLinkedRoleAsync(wg, errChans[i], roleName)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking IAM Service Linked Role",
			}, map[string]interface{}{})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return gruntworkerrors.WithStackTrace(finalErr)
	}

	for _, roleName := range roleNames {
		logging.Debugf("[OK] IAM Service Linked Role %s was deleted.", aws.StringValue(roleName))
	}
	return nil
}

func (islr *IAMServiceLinkedRoles) shouldInclude(iamServiceLinkedRole *iam.Role, configObj config.Config) bool {
	if !strings.Contains(aws.StringValue(iamServiceLinkedRole.Arn), "aws-service-role") {
		return false
	}

	return configObj.IAMServiceLinkedRoles.ShouldInclude(config.ResourceValue{
		Time: iamServiceLinkedRole.CreateDate,
		Name: iamServiceLinkedRole.RoleName,
	})
}

func (islr *IAMServiceLinkedRoles) deleteIamServiceLinkedRoleAsync(wg *sync.WaitGroup, errChan chan error, roleName *string) {
	defer wg.Done()

	var result *multierror.Error

	// Functions used to really nuke an IAM Role as a role can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual role deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(roleName *string) error{
		islr.deleteIamServiceLinkedRole,
	}

	for _, fn := range functions {
		if err := fn(roleName); err != nil {
			result = multierror.Append(result, err)
		}
	}

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(roleName),
		ResourceType: "IAM Service Linked Role",
		Error:        result.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- result.ErrorOrNil()
}

// Custom errors

type TooManyIamServiceLinkedRoleErr struct{}

func (err TooManyIamServiceLinkedRoleErr) Error() string {
	return "Too many IAM Service Linked Roles requested at once"
}
