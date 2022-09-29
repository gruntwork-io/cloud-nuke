package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllIamGroups(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	allIamGroups := []*string{}
	err := svc.ListGroupsPages(
		&iam.ListGroupsInput{},
		func(page *iam.ListGroupsOutput, lastPage bool) bool {
			for _, iamGroup := range page.Groups {
				if shouldIncludeIamGroup(iamGroup, excludeAfter, configObj) {
					allIamGroups = append(allIamGroups, iamGroup.GroupName)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return allIamGroups, nil
}

//nukeAllIamGroups - delete all IAM Roles.  Caller is responsible for pagination (no more than 100/request)
func nukeAllIamGroups(session *session.Session, groupNames []*string) error {
	region := aws.StringValue(session.Config.Region) //Since this is a global resource this can be any random region
	svc := iam.New(session)

	if len(groupNames) == 0 {
		logging.Logger.Info("No IAM Groups to nuke")
		return nil
	}

	//Probably not required since pagination is handled by the caller
	if len(groupNames) > 100 {
		logging.Logger.Errorf("Nuking too many IAM Groups at once (100): Halting to avoid rate limits")
		return TooManyIamGroupErr{}
	}

	//No bulk delete exists, do it with goroutines
	//TODO

	//TODO implement
	return nil
}

//deleteIamGroup - removes an IAM group from AWS
func deleteIamGroup(svc *iam.IAM, groupName *string) error {
	_, err := svc.DeleteGroup(&iam.DeleteGroupInput{
		GroupName: groupName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

//check if iam group should be included based on config rules (RegExp and Exclude After)
func shouldIncludeIamGroup(iamGroup *iam.Group, excludeAfter time.Time, configObj config.Config) bool {
	if iamGroup == nil {
		return false
	}

	if excludeAfter.Before(*iamGroup.CreateDate) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(iamGroup.GroupName),
		configObj.IAMGroups.IncludeRule.NamesRegExp,
		configObj.IAMGroups.ExcludeRule.NamesRegExp,
	)
}

//TODO delete policy functions belong here eventually but out of scope for trial

//Custom Errors
type TooManyIamGroupErr struct{}

func (err TooManyIamGroupErr) Error() string {
	return "Too many IAM Groups requested at once"
}

//Sanity Check

// 1. aws.go lists all the resources
// 2. I'd be adding something to the global resources that checks for any nukeable groups
// 3. If not dry run, aws.go calls the .Nuke() function on each resource
// 4. As far as I can tell (excluding potentially policies) there are no pre-requisites for removing
//		an iam group.  Any users would get cleaned up by existing functionality and order shouldn't matter
// 5. I'll add IAMGroups to the config.go file which may get us config file support for free, not 100% sure

//IAM GROUP TYPES
//		Implement AwsResources interface
