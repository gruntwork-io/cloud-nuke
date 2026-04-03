package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

const (
	// permanentDeletionDays is the number of days before a CA is permanently deleted.
	// AWS allows 7-30 days; we use the minimum since cloud-nuke targets non-production resources.
	permanentDeletionDays = 7
)

// ACMPCAAPI defines the interface for ACM PCA operations.
type ACMPCAAPI interface {
	DeleteCertificateAuthority(ctx context.Context, params *acmpca.DeleteCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DeleteCertificateAuthorityOutput, error)
	DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error)
	ListCertificateAuthorities(ctx context.Context, params *acmpca.ListCertificateAuthoritiesInput, optFns ...func(*acmpca.Options)) (*acmpca.ListCertificateAuthoritiesOutput, error)
	ListTags(ctx context.Context, params *acmpca.ListTagsInput, optFns ...func(*acmpca.Options)) (*acmpca.ListTagsOutput, error)
	UpdateCertificateAuthority(ctx context.Context, params *acmpca.UpdateCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.UpdateCertificateAuthorityOutput, error)
}

// NewACMPCA creates a new ACMPCA resource using the generic resource pattern.
func NewACMPCA() AwsResource {
	return NewAwsResource(&resource.Resource[ACMPCAAPI]{
		ResourceTypeName: "acmpca",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ACMPCAAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = acmpca.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ACMPCA
		},
		Lister: listACMPCA,
		Nuker:  resource.SimpleBatchDeleter(deleteACMPCA),
	})
}

// listACMPCA retrieves all ACM PCA certificate authorities that match the config filters.
func listACMPCA(ctx context.Context, client ACMPCAAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var arns []*string

	paginator := acmpca.NewListCertificateAuthoritiesPaginator(client, &acmpca.ListCertificateAuthoritiesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, ca := range page.CertificateAuthorities {
			if ca.Status == types.CertificateAuthorityStatusDeleted {
				continue
			}

			referenceTime := aws.ToTime(ca.LastStateChangeAt)
			if ca.LastStateChangeAt == nil {
				referenceTime = aws.ToTime(ca.CreatedAt)
			}

			var tags map[string]string
			tagsOutput, err := client.ListTags(ctx, &acmpca.ListTagsInput{
				CertificateAuthorityArn: ca.Arn,
			})
			if err != nil {
				logging.Debugf("Error getting tags for ACMPCA %s: %v", aws.ToString(ca.Arn), err)
				continue
			}
			if tagsOutput != nil {
				tags = util.ConvertACMPCATagsToMap(tagsOutput.Tags)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: &referenceTime,
				Tags: tags,
			}) {
				arns = append(arns, ca.Arn)
			}
		}
	}

	return arns, nil
}

// deleteACMPCA deletes a single ACM PCA certificate authority.
// This function handles the multi-step deletion process:
// 1. Describe the CA to get its current status
// 2. Disable the CA if it's in ACTIVE state (AWS requires this before deletion)
// 3. Delete the CA with a 7-day restoration period
func deleteACMPCA(ctx context.Context, client ACMPCAAPI, arn *string) error {
	arnStr := aws.ToString(arn)

	logging.Debugf("Fetching CA details for ACMPCA %s", arnStr)
	details, err := client.DescribeCertificateAuthority(ctx, &acmpca.DescribeCertificateAuthorityInput{
		CertificateAuthorityArn: arn,
	})
	if err != nil {
		return fmt.Errorf("failed to describe ACMPCA %s: %w", arnStr, err)
	}
	if details.CertificateAuthority == nil {
		return fmt.Errorf("CA not found: %s", arnStr)
	}
	if details.CertificateAuthority.Status == "" {
		return fmt.Errorf("CA status unavailable: %s", arnStr)
	}

	// AWS requires ACTIVE CAs to be disabled before deletion.
	// CAs in CREATING, PENDING_CERTIFICATE, DISABLED, or DELETED states can be deleted directly.
	if needsDisableBeforeDelete(details.CertificateAuthority.Status) {
		logging.Debugf("Disabling ACMPCA %s before deletion", arnStr)
		if _, err = client.UpdateCertificateAuthority(ctx, &acmpca.UpdateCertificateAuthorityInput{
			CertificateAuthorityArn: arn,
			Status:                  types.CertificateAuthorityStatusDisabled,
		}); err != nil {
			return fmt.Errorf("failed to disable ACMPCA %s: %w", arnStr, err)
		}
	}

	_, err = client.DeleteCertificateAuthority(ctx, &acmpca.DeleteCertificateAuthorityInput{
		CertificateAuthorityArn:     arn,
		PermanentDeletionTimeInDays: aws.Int32(permanentDeletionDays),
	})
	if err != nil {
		return fmt.Errorf("failed to delete ACMPCA %s: %w", arnStr, err)
	}

	logging.Debugf("Deleted ACMPCA: %s", arnStr)
	return nil
}

// needsDisableBeforeDelete returns true if the CA must be disabled before deletion.
func needsDisableBeforeDelete(status types.CertificateAuthorityStatus) bool {
	switch status {
	case types.CertificateAuthorityStatusCreating,
		types.CertificateAuthorityStatusPendingCertificate,
		types.CertificateAuthorityStatusDisabled,
		types.CertificateAuthorityStatusDeleted:
		return false
	default:
		// ACTIVE, EXPIRED, FAILED states require disable first
		return true
	}
}
