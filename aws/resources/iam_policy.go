package resources

import (
	"context"
	"github.com/gruntwork-io/cloud-nuke/util"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// Returns the ARN of all customer managed policies
func (ip *IAMPolicies) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allIamPolicies []*string
	paginator := iam.NewListPoliciesPaginator(ip.Client, &iam.ListPoliciesInput{Scope: types.PolicyScopeTypeLocal})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, policy := range page.Policies {
			// Always fetch tags to support tag-based filtering, including the default cloud-nuke-excluded tag.
			// This ensures that policies with the exclusion tag are properly filtered out even when no explicit
			// tag filters are configured in the config file.
			tagsOut, err := ip.Client.ListPolicyTags(c, &iam.ListPolicyTagsInput{PolicyArn: policy.Arn})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			tags := tagsOut.Tags

			if configObj.IAMPolicies.ShouldInclude(config.ResourceValue{
				Name: policy.PolicyName,
				Time: policy.CreateDate,
				Tags: util.ConvertIAMTagsToMap(tags),
			}) {
				allIamPolicies = append(allIamPolicies, policy.Arn)
			}
		}
	}

	return allIamPolicies, nil
}

// Delete all iam customer managed policies. Caller is responsible for pagination (no more than 100/request)
func (ip *IAMPolicies) nukeAll(policyArns []*string) error {
	if len(policyArns) == 0 {
		logging.Debug("No IAM Policies to nuke")
	}

	// Probably not required since pagination is handled by the caller
	if len(policyArns) > 100 {
		logging.Errorf("Nuking too many IAM Policies at once (100): Halting to avoid rate limits")
		return TooManyIamPolicyErr{}
	}

	// No Bulk Delete exists, do it with goroutines
	logging.Debug("Deleting all IAM Policies")
	wg := new(sync.WaitGroup)
	wg.Add(len(policyArns))
	errChans := make([]chan error, len(policyArns))
	for i, arn := range policyArns {
		errChans[i] = make(chan error, 1)
		go ip.deleteIamPolicyAsync(wg, errChans[i], arn)
	}
	wg.Wait()

	// Collapse the errors down to one
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Errorf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	return nil
}

// Removes an IAM Policy from AWS, designed to run as a goroutine
func (ip *IAMPolicies) deleteIamPolicyAsync(wg *sync.WaitGroup, errChan chan error, policyArn *string) {
	defer wg.Done()
	var multierr *multierror.Error

	// Detach any entities the policy is attached to
	err := ip.detachPolicyEntities(policyArn)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	}

	// Get Old Policy Versions
	var versionsToRemove []*string
	paginator := iam.NewListPolicyVersionsPaginator(ip.Client, &iam.ListPolicyVersionsInput{PolicyArn: policyArn})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			multierr = multierror.Append(multierr, err)
			break
		}

		for _, policyVersion := range page.Versions {
			if !policyVersion.IsDefaultVersion {
				versionsToRemove = append(versionsToRemove, policyVersion.VersionId)
			}
		}
	}

	// Delete old policy versions
	for _, versionId := range versionsToRemove {
		_, err = ip.Client.DeletePolicyVersion(ip.Context, &iam.DeletePolicyVersionInput{VersionId: versionId, PolicyArn: policyArn})
		if err != nil {
			multierr = multierror.Append(multierr, err)
		}
	}
	// Delete the policy
	_, err = ip.Client.DeletePolicy(ip.Context, &iam.DeletePolicyInput{PolicyArn: policyArn})
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		logging.Debugf("[OK] IAM Policy %s was deleted in global", aws.ToString(policyArn))
	}

	e := report.Entry{
		Identifier:   aws.ToString(policyArn),
		ResourceType: "IAM Policy",
		Error:        multierr.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- multierr.ErrorOrNil()
}

func (ip *IAMPolicies) detachPolicyEntities(policyArn *string) error {
	var allPolicyGroups []*string
	var allPolicyRoles []*string
	var allPolicyUsers []*string

	paginator := iam.NewListEntitiesForPolicyPaginator(ip.Client, &iam.ListEntitiesForPolicyInput{PolicyArn: policyArn})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ip.Context)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, group := range page.PolicyGroups {
			allPolicyGroups = append(allPolicyGroups, group.GroupName)
		}
		for _, role := range page.PolicyRoles {
			allPolicyRoles = append(allPolicyRoles, role.RoleName)
		}
		for _, user := range page.PolicyUsers {
			allPolicyUsers = append(allPolicyUsers, user.UserName)
		}
	}

	// Detach policy from any users
	for _, userName := range allPolicyUsers {
		detachUserInput := &iam.DetachUserPolicyInput{
			UserName:  userName,
			PolicyArn: policyArn,
		}
		_, err := ip.Client.DetachUserPolicy(ip.Context, detachUserInput)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}
	// Detach policy from any groups
	for _, groupName := range allPolicyGroups {
		detachGroupInput := &iam.DetachGroupPolicyInput{
			GroupName: groupName,
			PolicyArn: policyArn,
		}
		_, err := ip.Client.DetachGroupPolicy(ip.Context, detachGroupInput)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}
	// Detach policy from any roles
	for _, roleName := range allPolicyRoles {
		detachRoleInput := &iam.DetachRolePolicyInput{
			RoleName:  roleName,
			PolicyArn: policyArn,
		}
		_, err := ip.Client.DetachRolePolicy(ip.Context, detachRoleInput)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

// TooManyIamPolicyErr Custom Errors
type TooManyIamPolicyErr struct{}

func (err TooManyIamPolicyErr) Error() string {
	return "Too many IAM Groups requested at once"
}
