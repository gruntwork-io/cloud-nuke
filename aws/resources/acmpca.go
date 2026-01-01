package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ACMPCAAPI defines the interface for ACM PCA operations.
type ACMPCAAPI interface {
	DeleteCertificateAuthority(ctx context.Context, params *acmpca.DeleteCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DeleteCertificateAuthorityOutput, error)
	DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error)
	ListCertificateAuthorities(ctx context.Context, params *acmpca.ListCertificateAuthoritiesInput, optFns ...func(*acmpca.Options)) (*acmpca.ListCertificateAuthoritiesOutput, error)
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
			if shouldIncludeACMPCA(ca, cfg) {
				arns = append(arns, ca.Arn)
			}
		}
	}

	return arns, nil
}

// shouldIncludeACMPCA determines if an ACM PCA should be included based on config filters.
func shouldIncludeACMPCA(ca types.CertificateAuthority, cfg config.ResourceType) bool {
	statusSafe := ca.Status
	if statusSafe == types.CertificateAuthorityStatusDeleted {
		return false
	}

	// reference time for excludeAfter is lastStateChangeAt time,
	// unless it was never changed and createAt time is used.
	var referenceTime time.Time
	if ca.LastStateChangeAt == nil {
		referenceTime = aws.ToTime(ca.CreatedAt)
	} else {
		referenceTime = aws.ToTime(ca.LastStateChangeAt)
	}

	return cfg.ShouldInclude(config.ResourceValue{Time: &referenceTime})
}

// deleteACMPCA deletes a single ACM PCA certificate authority.
// This function handles the multi-step deletion process:
// 1. Describe the CA to get its current status
// 2. Disable the CA if it's not already in a deletable state
// 3. Delete the CA with a 7-day waiting period
func deleteACMPCA(ctx context.Context, client ACMPCAAPI, arn *string) error {
	logging.Debugf("Fetching details of CA to be deleted for ACMPCA %s", aws.ToString(arn))
	details, err := client.DescribeCertificateAuthority(
		ctx,
		&acmpca.DescribeCertificateAuthorityInput{CertificateAuthorityArn: arn})
	if err != nil {
		return fmt.Errorf("failed to describe ACMPCA %s: %w", aws.ToString(arn), err)
	}
	if details.CertificateAuthority == nil {
		return fmt.Errorf("could not find CA %s", aws.ToString(arn))
	}
	if details.CertificateAuthority.Status == "" {
		return fmt.Errorf("could not fetch status for CA %s", aws.ToString(arn))
	}

	// find out whether we have to disable the CA first, prior to deletion.
	statusSafe := details.CertificateAuthority.Status
	shouldUpdateStatus := statusSafe != types.CertificateAuthorityStatusCreating &&
		statusSafe != types.CertificateAuthorityStatusPendingCertificate &&
		statusSafe != types.CertificateAuthorityStatusDisabled &&
		statusSafe != types.CertificateAuthorityStatusDeleted

	if shouldUpdateStatus {
		logging.Debugf("Setting status to 'DISABLED' for ACMPCA %s", aws.ToString(arn))
		if _, err = client.UpdateCertificateAuthority(ctx, &acmpca.UpdateCertificateAuthorityInput{
			CertificateAuthorityArn: arn,
			Status:                  types.CertificateAuthorityStatusDisabled,
		}); err != nil {
			return fmt.Errorf("failed to disable ACMPCA %s: %w", aws.ToString(arn), err)
		}
		logging.Debugf("Did set status to 'DISABLED' for ACMPCA: %s", aws.ToString(arn))
	}

	_, err = client.DeleteCertificateAuthority(ctx, &acmpca.DeleteCertificateAuthorityInput{
		CertificateAuthorityArn: arn,
		// the range is 7 to 30 days.
		// since cloud-nuke should not be used in production,
		// we assume that the minimum (7 days) is fine.
		PermanentDeletionTimeInDays: aws.Int32(7),
	})
	if err != nil {
		return fmt.Errorf("failed to delete ACMPCA %s: %w", aws.ToString(arn), err)
	}

	logging.Debugf("Deleted ACMPCA: %s successfully", aws.ToString(arn))
	return nil
}
