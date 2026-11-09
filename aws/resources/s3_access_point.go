package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// S3ControlAccessPointAPI defines the interface for S3 Control Access Point operations.
type S3ControlAccessPointAPI interface {
	ListAccessPoints(ctx context.Context, params *s3control.ListAccessPointsInput, optFns ...func(*s3control.Options)) (*s3control.ListAccessPointsOutput, error)
	GetAccessPoint(ctx context.Context, params *s3control.GetAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.GetAccessPointOutput, error)
	DeleteAccessPoint(ctx context.Context, params *s3control.DeleteAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointOutput, error)
}

// NewS3AccessPoints creates a new S3 Access Point resource using the generic resource pattern.
func NewS3AccessPoints() AwsResource {
	lister, nuker := keepAccountID(listS3AccessPoints, nukeS3AccessPoints)
	return NewAwsResource(&resource.Resource[S3ControlAccessPointAPI]{
		ResourceTypeName: "s3-access-point",
		// S3 Control API has tight rate limits; keep batch size low to avoid throttling.
		BatchSize: 5,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[S3ControlAccessPointAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = s3control.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.S3AccessPoint
		},
		Lister: lister,
		Nuker:  nuker,
	})
}

// listS3AccessPoints retrieves all S3 Access Points that match the config filters.
func listS3AccessPoints(ctx context.Context, client S3ControlAccessPointAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return nil, errors.WithStackTrace(fmt.Errorf("unable to lookup the account id"))
	}

	var identifiers []*string
	paginator := s3control.NewListAccessPointsPaginator(client, &s3control.ListAccessPointsInput{
		AccountId: aws.String(accountID),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, accessPoint := range page.AccessPointList {
			var createdAt *time.Time
			if cfg.HasTimeFilter() && passesNameAndTagRules(cfg, accessPoint.Name, nil) {
				createdAt = getS3AccessPointCreationTime(ctx, client, accountID, accessPoint.Name)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: accessPoint.Name,
				Time: createdAt,
			}) {
				identifiers = append(identifiers, accessPoint.Name)
			}
		}
	}

	return identifiers, nil
}

// getS3AccessPointCreationTime reads the creation time of an access point,
// which ListAccessPoints does not return. It returns nil, and warns, when the
// time cannot be read, since a time filter then cannot include the access point.
func getS3AccessPointCreationTime(ctx context.Context, client S3ControlAccessPointAPI, accountID string, name *string) *time.Time {
	output, err := client.GetAccessPoint(ctx, &s3control.GetAccessPointInput{
		AccountId: aws.String(accountID),
		Name:      name,
	})
	if err != nil {
		logging.Warnf("Unable to read the creation time of S3 access point %s, so a time filter cannot include it: %v", aws.ToString(name), err)
		return nil
	}
	return output.CreationDate
}

// nukeS3AccessPoints deletes S3 Access Points.
func nukeS3AccessPoints(ctx context.Context, client S3ControlAccessPointAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return []resource.NukeResult{{Error: fmt.Errorf("unable to lookup the account id")}}
	}

	deleteFn := func(ctx context.Context, client S3ControlAccessPointAPI, name *string) error {
		_, err := client.DeleteAccessPoint(ctx, &s3control.DeleteAccessPointInput{
			AccountId: aws.String(accountID),
			Name:      name,
		})
		return err
	}

	return resource.SimpleBatchDeleter(deleteFn)(ctx, client, scope, resourceType, identifiers)
}
