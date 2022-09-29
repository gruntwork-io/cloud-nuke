package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
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
	logging.Logger.Info("Deleting all IAM Groups")
	wg := new(sync.WaitGroup)
	wg.Add(len(groupNames))
	errChans := make([]chan error, len(groupNames))
	for i, groupName := range groupNames {
		errChans[i] = make(chan error, 1)
		go deleteIamGroupAsync(wg, errChans[i], svc, groupName)
	}
	wg.Wait()

	//Collapse the errors down to one
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	//Print Successful deletions
	for _, groupName := range groupNames {
		logging.Logger.Infof("[OK] IAM Group %s was deleted in %s", aws.StringValue(groupName), region)
	}
	return nil
}

//deleteIamGroup - removes an IAM group from AWS, designed to run as a goroutine
func deleteIamGroupAsync(wg *sync.WaitGroup, errChan chan error, svc *iam.IAM, groupName *string) {
	defer wg.Done()
	_, err := svc.DeleteGroup(&iam.DeleteGroupInput{
		GroupName: groupName,
	})
	errChan <- err
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

//TODO delete policy functions belong here eventually but out of scope for now

//Custom Errors
type TooManyIamGroupErr struct{}

func (err TooManyIamGroupErr) Error() string {
	return "Too many IAM Groups requested at once"
}
