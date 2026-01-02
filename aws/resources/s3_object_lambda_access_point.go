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

// S3ObjectLambdaAccessPointAPI defines the interface for S3 Object Lambda Access Point operations.
type S3ObjectLambdaAccessPointAPI interface {
	ListAccessPointsForObjectLambda(ctx context.Context, params *s3control.ListAccessPointsForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.ListAccessPointsForObjectLambdaOutput, error)
	DeleteAccessPointForObjectLambda(ctx context.Context, params *s3control.DeleteAccessPointForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointForObjectLambdaOutput, error)
}

// NewS3ObjectLambdaAccessPoints creates a new S3 Object Lambda Access Point resource.
func NewS3ObjectLambdaAccessPoints() AwsResource {
	return NewAwsResource(&resource.Resource[S3ObjectLambdaAccessPointAPI]{
		ResourceTypeName: "s3-olap",
		BatchSize:        5,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[S3ObjectLambdaAccessPointAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = s3control.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.S3ObjectLambdaAccessPoint
		},
		Lister: listS3ObjectLambdaAccessPoints,
		Nuker:  nukeS3ObjectLambdaAccessPoints,
	})
}

// listS3ObjectLambdaAccessPoints retrieves all Object Lambda Access Points that match the config filters.
func listS3ObjectLambdaAccessPoints(ctx context.Context, client S3ObjectLambdaAccessPointAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return nil, errors.WithStackTrace(fmt.Errorf("unable to lookup the account id"))
	}

	var identifiers []*string
	paginator := s3control.NewListAccessPointsForObjectLambdaPaginator(client, &s3control.ListAccessPointsForObjectLambdaInput{
		AccountId: aws.String(accountID),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, accessPoint := range page.ObjectLambdaAccessPointList {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: accessPoint.Name,
			}) {
				identifiers = append(identifiers, accessPoint.Name)
			}
		}
	}

	return identifiers, nil
}

// nukeS3ObjectLambdaAccessPoints deletes Object Lambda Access Points.
func nukeS3ObjectLambdaAccessPoints(ctx context.Context, client S3ObjectLambdaAccessPointAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		logging.Errorf("unable to read the account-id from context")
		return []resource.NukeResult{{Error: fmt.Errorf("unable to lookup the account id")}}
	}

	deleteFn := func(ctx context.Context, client S3ObjectLambdaAccessPointAPI, name *string) error {
		_, err := client.DeleteAccessPointForObjectLambda(ctx, &s3control.DeleteAccessPointForObjectLambdaInput{
			AccountId: aws.String(accountID),
			Name:      name,
		})
		return err
	}

	return resource.SimpleBatchDeleter(deleteFn)(ctx, client, scope, resourceType, identifiers)
}
