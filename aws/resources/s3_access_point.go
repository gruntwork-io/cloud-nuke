package resources

import (
	"context"
	"fmt"

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
	DeleteAccessPoint(ctx context.Context, params *s3control.DeleteAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointOutput, error)
}

// NewS3AccessPoints creates a new S3 Access Point resource using the generic resource pattern.
func NewS3AccessPoints() AwsResource {
	return NewAwsResource(&resource.Resource[S3ControlAccessPointAPI]{
		ResourceTypeName: "s3-ap",
		BatchSize:        5,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[S3ControlAccessPointAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = s3control.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.S3AccessPoint
		},
		Lister: listS3AccessPoints,
		Nuker:  nukeS3AccessPoints,
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
			if cfg.ShouldInclude(config.ResourceValue{
				Name: accessPoint.Name,
			}) {
				identifiers = append(identifiers, accessPoint.Name)
			}
		}
	}

	return identifiers, nil
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
