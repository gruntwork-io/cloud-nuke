package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (ap *S3AccessPoint) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	accountID, ok := c.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return nil, errors.WithStackTrace(fmt.Errorf("unable to lookup the account id"))
	}

	// set the account id in object as this is mandatory to nuke an access point
	ap.AccountID = aws.String(accountID)

	var accessPoints []*string
	paginator := s3control.NewListAccessPointsPaginator(ap.Client, &s3control.ListAccessPointsInput{
		AccountId: ap.AccountID,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ap.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, accessPoint := range page.AccessPointList {
			if configObj.S3AccessPoint.ShouldInclude(config.ResourceValue{
				Name: accessPoint.Name,
			}) {
				accessPoints = append(accessPoints, accessPoint.Name)
			}
		}
	}

	return accessPoints, nil
}

func (ap *S3AccessPoint) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Access point(s) to nuke in region %s", ap.Region)
		return nil
	}

	logging.Debugf("Deleting all Access points in region %s", ap.Region)
	var deleted []*string

	for _, id := range identifiers {

		_, err := ap.Client.DeleteAccessPoint(ap.Context, &s3control.DeleteAccessPointInput{
			AccountId: ap.AccountID,
			Name:      id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "S3 Access point",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
			logging.Debugf("Deleted S3 access point: %s", aws.ToString(id))
		}
	}

	logging.Debugf("[OK] %d S3 Access point(s) deleted in %s", len(deleted), ap.Region)

	return nil
}
