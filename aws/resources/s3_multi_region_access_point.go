package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// S3ControlMultiRegionAPI defines the interface for S3 Multi-Region Access Point operations.
type S3ControlMultiRegionAPI interface {
	ListMultiRegionAccessPoints(ctx context.Context, params *s3control.ListMultiRegionAccessPointsInput, optFns ...func(*s3control.Options)) (*s3control.ListMultiRegionAccessPointsOutput, error)
	DeleteMultiRegionAccessPoint(ctx context.Context, params *s3control.DeleteMultiRegionAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteMultiRegionAccessPointOutput, error)
}

// NewS3MultiRegionAccessPoints creates a new S3 Multi-Region Access Point resource.
func NewS3MultiRegionAccessPoints() AwsResource {
	return NewAwsResource(&resource.Resource[S3ControlMultiRegionAPI]{
		ResourceTypeName: "s3-mrap",
		BatchSize:        5,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[S3ControlMultiRegionAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = s3control.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.S3MultiRegionAccessPoint
		},
		Lister: listS3MultiRegionAccessPoints,
		Nuker:  nukeS3MultiRegionAccessPoints,
	})
}

// listS3MultiRegionAccessPoints retrieves all S3 Multi-Region Access Points that match the config filters.
//
// NOTE: All control plane requests to create or maintain Multi-Region Access Points must be routed to US West (Oregon) Region.
// Reference: https://docs.aws.amazon.com/AmazonS3/latest/userguide/MultiRegionAccessPointRestrictions.html
func listS3MultiRegionAccessPoints(ctx context.Context, client S3ControlMultiRegionAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return nil, errors.WithStackTrace(fmt.Errorf("unable to lookup the account id"))
	}

	var identifiers []*string
	paginator := s3control.NewListMultiRegionAccessPointsPaginator(client, &s3control.ListMultiRegionAccessPointsInput{
		AccountId: aws.String(accountID),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, accessPoint := range page.AccessPoints {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: accessPoint.Name,
				Time: accessPoint.CreatedAt,
			}) {
				identifiers = append(identifiers, accessPoint.Name)
			}
		}
	}

	return identifiers, nil
}

// nukeS3MultiRegionAccessPoints deletes S3 Multi-Region Access Points.
func nukeS3MultiRegionAccessPoints(ctx context.Context, client S3ControlMultiRegionAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return []resource.NukeResult{{Error: fmt.Errorf("unable to lookup the account id")}}
	}

	deleteFn := func(ctx context.Context, client S3ControlMultiRegionAPI, name *string) error {
		_, err := client.DeleteMultiRegionAccessPoint(ctx, &s3control.DeleteMultiRegionAccessPointInput{
			AccountId: aws.String(accountID),
			Details: &types.DeleteMultiRegionAccessPointInput{
				Name: name,
			},
		})
		return err
	}

	return resource.SequentialDeleter(deleteFn)(ctx, client, scope, resourceType, identifiers)
}
