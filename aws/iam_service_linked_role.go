package aws

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// List all IAM Roles in the AWS account
func getAllIamServiceLinkedRoles(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	allIAMServiceLinkedRoles := []*string{}
	err := svc.ListRolesPages(
		&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, iamServiceLinkedRole := range page.Roles {
				if shouldIncludeIAMServiceLinkedRole(iamServiceLinkedRole, excludeAfter, configObj) {
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

func deleteIamServiceLinkedRole(svc *iam.IAM, roleName *string) error {
	// Deletion ID looks like this: "
	//{
	//	DeletionTaskId: "task/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling_2/d3c4c9fc-7fd3-4a36-974a-afb0eb78f102"
	//}
	deletionData, err := svc.DeleteServiceLinkedRole(&iam.DeleteServiceLinkedRoleInput{
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
		deletionStatus, err = svc.GetServiceLinkedRoleDeletionStatus(&iam.GetServiceLinkedRoleDeletionStatusInput{
			DeletionTaskId: deletionData.DeletionTaskId,
		})
		if err != nil {
			return gruntworkerrors.WithStackTrace(err)
		}
		if aws.StringValue(deletionStatus.Status) == "IN_PROGRESS" {
			logging.Logger.Debugf("Deletion of IAM ServiceLinked Role %s is still in progress", aws.StringValue(roleName))
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
func nukeAllIamServiceLinkedRoles(session *session.Session, roleNames []*string) error {
	region := aws.StringValue(session.Config.Region)
	svc := iam.New(session)

	if len(roleNames) == 0 {
		logging.Logger.Debug("No IAM Service Linked Roles to nuke")
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on IAMRole.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(roleNames) > 100 {
		logging.Logger.Debugf("Nuking too many IAM Service Linked Roles at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyIamRoleErr{}
	}

	// There is no bulk delete IAM Roles API, so we delete the batch of IAM roles concurrently using go routines
	logging.Logger.Debugf("Deleting all IAM Service Linked Roles in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(roleNames))
	errChans := make([]chan error, len(roleNames))
	for i, roleName := range roleNames {
		errChans[i] = make(chan error, 1)
		go deleteIamServiceLinkedRoleAsync(wg, errChans[i], svc, roleName)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking IAM Service Linked Role",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return gruntworkerrors.WithStackTrace(finalErr)
	}

	for _, roleName := range roleNames {
		logging.Logger.Debugf("[OK] IAM Service Linked Role %s was deleted in %s", aws.StringValue(roleName), region)
	}
	return nil
}

func shouldIncludeIAMServiceLinkedRole(iamServiceLinkedRole *iam.Role, excludeAfter time.Time, configObj config.Config) bool {
	if iamServiceLinkedRole == nil {
		return false
	}

	if !strings.Contains(aws.StringValue(iamServiceLinkedRole.Arn), "aws-service-role") {
		return false
	}

	if excludeAfter.Before(*iamServiceLinkedRole.CreateDate) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(iamServiceLinkedRole.RoleName),
		configObj.IAMServiceLinkedRole.IncludeRule.NamesRegExp,
		configObj.IAMServiceLinkedRole.ExcludeRule.NamesRegExp,
	)
}

func deleteIamServiceLinkedRoleAsync(wg *sync.WaitGroup, errChan chan error, svc *iam.IAM, roleName *string) {
	defer wg.Done()

	var result *multierror.Error

	// Functions used to really nuke an IAM Role as a role can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual role deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(svc *iam.IAM, roleName *string) error{
		deleteIamServiceLinkedRole,
	}

	for _, fn := range functions {
		if err := fn(svc, roleName); err != nil {
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
