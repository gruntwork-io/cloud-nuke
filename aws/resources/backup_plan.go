package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// BackupPlanAPI defines the interface for Backup Plan operations.
type BackupPlanAPI interface {
	DeleteBackupPlan(ctx context.Context, params *backup.DeleteBackupPlanInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupPlanOutput, error)
	DeleteBackupSelection(ctx context.Context, params *backup.DeleteBackupSelectionInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupSelectionOutput, error)
	ListBackupPlans(ctx context.Context, params *backup.ListBackupPlansInput, optFns ...func(*backup.Options)) (*backup.ListBackupPlansOutput, error)
	ListBackupSelections(ctx context.Context, params *backup.ListBackupSelectionsInput, optFns ...func(*backup.Options)) (*backup.ListBackupSelectionsOutput, error)
	ListTags(ctx context.Context, params *backup.ListTagsInput, optFns ...func(*backup.Options)) (*backup.ListTagsOutput, error)
}

// NewBackupPlan creates a new BackupPlan resource using the generic resource pattern.
func NewBackupPlan() AwsResource {
	return NewAwsResource(&resource.Resource[BackupPlanAPI]{
		ResourceTypeName: "backup-plan",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[BackupPlanAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = backup.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.BackupPlan
		},
		Lister: listBackupPlans,
		Nuker:  resource.MultiStepDeleter(nukeBackupSelections, nukeBackupPlan),
	})
}

// listBackupPlans retrieves all Backup Plans that match the config filters.
func listBackupPlans(ctx context.Context, client BackupPlanAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string
	paginator := backup.NewListBackupPlansPaginator(client, &backup.ListBackupPlansInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, plan := range page.BackupPlansList {
			tags, err := getBackupPlanTags(ctx, client, cfg, plan)
			if err != nil {
				logging.Errorf("Unable to fetch tags for %s: %s", aws.ToString(plan.BackupPlanArn), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: plan.BackupPlanName,
				Time: plan.CreationDate,
				Tags: tags,
			}) {
				ids = append(ids, plan.BackupPlanId)
			}
		}
	}

	return ids, nil
}

// getBackupPlanTags retrieves the tags for a given backup plan if tag-based filters are specified in the config.
func getBackupPlanTags(ctx context.Context, client BackupPlanAPI, cfg config.ResourceType, plan types.BackupPlansListMember) (map[string]string, error) {
	tags := map[string]string{}
	if len(cfg.IncludeRule.Tags) > 0 || len(cfg.ExcludeRule.Tags) > 0 {
		tagsPaginator := backup.NewListTagsPaginator(client, &backup.ListTagsInput{
			ResourceArn: plan.BackupPlanArn,
		})
		for tagsPaginator.HasMorePages() {
			tagsPage, err := tagsPaginator.NextPage(ctx)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			for tagKey, tagValue := range tagsPage.Tags {
				tags[tagKey] = tagValue
			}
		}
	}
	return tags, nil
}

// nukeBackupSelections deletes all backup selections for a backup plan.
func nukeBackupSelections(ctx context.Context, client BackupPlanAPI, id *string) error {
	planId := aws.ToString(id)
	logging.Debugf("Nuking backup selections of backup plan %s", planId)

	paginator := backup.NewListBackupSelectionsPaginator(client, &backup.ListBackupSelectionsInput{
		BackupPlanId: id,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Failed] listing backup selections of backup plan %s: %v", planId, err)
			return err
		}

		for _, selection := range page.BackupSelectionsList {
			selectionId := aws.ToString(selection.SelectionId)
			logging.Debugf("Deleting backup selection %s from backup plan %s", selectionId, planId)

			_, err = client.DeleteBackupSelection(ctx, &backup.DeleteBackupSelectionInput{
				BackupPlanId: id,
				SelectionId:  selection.SelectionId,
			})
			if err != nil {
				logging.Debugf("[Failed] deleting backup selection %s: %v", selectionId, err)
				return err
			}
		}
	}

	logging.Debugf("[OK] Successfully nuked backup selections of backup plan %s", planId)
	return nil
}

// nukeBackupPlan deletes a single backup plan.
func nukeBackupPlan(ctx context.Context, client BackupPlanAPI, id *string) error {
	_, err := client.DeleteBackupPlan(ctx, &backup.DeleteBackupPlanInput{
		BackupPlanId: id,
	})
	if err != nil {
		logging.Debugf("[Failed] nuking the backup plan %s: %v", aws.ToString(id), err)
		return err
	}
	return nil
}
