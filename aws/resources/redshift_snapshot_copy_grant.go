package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// RedshiftSnapshotCopyGrantsAPI defines the interface for Redshift Snapshot Copy Grant operations.
type RedshiftSnapshotCopyGrantsAPI interface {
	DescribeSnapshotCopyGrants(ctx context.Context, params *redshift.DescribeSnapshotCopyGrantsInput, optFns ...func(*redshift.Options)) (*redshift.DescribeSnapshotCopyGrantsOutput, error)
	DeleteSnapshotCopyGrant(ctx context.Context, params *redshift.DeleteSnapshotCopyGrantInput, optFns ...func(*redshift.Options)) (*redshift.DeleteSnapshotCopyGrantOutput, error)
}

// NewRedshiftSnapshotCopyGrants creates a new Redshift Snapshot Copy Grants resource.
func NewRedshiftSnapshotCopyGrants() AwsResource {
	return NewAwsResource(&resource.Resource[RedshiftSnapshotCopyGrantsAPI]{
		ResourceTypeName: "redshift-snapshot-copy-grant",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[RedshiftSnapshotCopyGrantsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = redshift.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.RedshiftSnapshotCopyGrant
		},
		Lister: listRedshiftSnapshotCopyGrants,
		Nuker:  resource.SimpleBatchDeleter(deleteRedshiftSnapshotCopyGrant),
	})
}

// listRedshiftSnapshotCopyGrants retrieves all Redshift Snapshot Copy Grants that match the config filters.
func listRedshiftSnapshotCopyGrants(ctx context.Context, client RedshiftSnapshotCopyGrantsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := redshift.NewDescribeSnapshotCopyGrantsPaginator(client, &redshift.DescribeSnapshotCopyGrantsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, grant := range page.SnapshotCopyGrants {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: grant.SnapshotCopyGrantName,
			}) {
				identifiers = append(identifiers, grant.SnapshotCopyGrantName)
			}
		}
	}

	return identifiers, nil
}

// deleteRedshiftSnapshotCopyGrant deletes a single Redshift Snapshot Copy Grant.
func deleteRedshiftSnapshotCopyGrant(ctx context.Context, client RedshiftSnapshotCopyGrantsAPI, id *string) error {
	_, err := client.DeleteSnapshotCopyGrant(ctx, &redshift.DeleteSnapshotCopyGrantInput{
		SnapshotCopyGrantName: id,
	})
	return err
}
