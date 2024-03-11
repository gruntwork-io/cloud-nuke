package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (ap *S3ObjectLambdaAccessPoint) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	accountID, ok := c.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return nil, errors.WithStackTrace(fmt.Errorf("unable to lookup the account id"))
	}

	// set the account id in object as this is mandatory to nuke an access point
	ap.AccountID = aws.String(accountID)

	var accessPoints []*string
	err := ap.Client.ListAccessPointsForObjectLambdaPages(&s3control.ListAccessPointsForObjectLambdaInput{
		AccountId: ap.AccountID,
	}, func(lapo *s3control.ListAccessPointsForObjectLambdaOutput, lastPage bool) bool {
		for _, accessPoint := range lapo.ObjectLambdaAccessPointList {
			if configObj.S3ObjectLambdaAccessPoint.ShouldInclude(config.ResourceValue{
				Name: accessPoint.Name,
			}) {
				accessPoints = append(accessPoints, accessPoint.Name)
			}
		}
		return !lastPage
	})
	return accessPoints, errors.WithStackTrace(err)
}

func (ap *S3ObjectLambdaAccessPoint) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Object lambda access point(s) to nuke in region %s", ap.Region)
		return nil
	}

	logging.Debugf("Deleting all Object lambda access points in region %s", ap.Region)
	var deleted []*string

	for _, id := range identifiers {

		_, err := ap.Client.DeleteAccessPointForObjectLambda(&s3control.DeleteAccessPointForObjectLambdaInput{
			AccountId: ap.AccountID,
			Name:      id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "S3 Object Lambda Access point",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
			logging.Debugf("Deleted S3 object lambda access point: %s", aws.StringValue(id))
		}
	}

	logging.Debugf("[OK] %d S3 Object lambda access point(s) deleted in %s", len(deleted), ap.Region)

	return nil
}
