package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func getAllIamGroups(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	//TODO implement
	return nil, nil
}

//nukeAllIamGroups - delete all IAM Roles
func nukeAllIamGroups(session *session.Session, groupNames []*string) error {
	//TODO implement
	return nil
}

//deleteIamGroupAsync - Asynchronosly remove an IAM group from AWS along with any sub-items
func deleteIamGroupAsync() {
	//TODO implement
}

//TODO do I need shouldInclude function?

//deleteIamGroup - removes an IAM group from AWS (ensure group is empty before calling?)
func deleteIamGroup(svc *iam.IAM, groupName *string) error {
	return nil
}

//Not sure if needed
func shouldIncludeIamGroup(iamGroup *iam.Group, excludeAfter time.Time, configObj config.Config) bool {
	return false
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
