package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ACMAPI defines the interface for ACM operations.
type ACMAPI interface {
	ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
	DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
}

// NewACM creates a new ACM resource using the generic resource pattern.
func NewACM() AwsResource {
	return NewAwsResource(&resource.Resource[ACMAPI]{
		ResourceTypeName: "acm",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ACMAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = acm.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ACM
		},
		Lister: listACMCertificates,
		Nuker:  resource.SimpleBatchDeleter(deleteACMCertificate),
	})
}

// listACMCertificates retrieves all ACM certificates that match the config filters.
func listACMCertificates(ctx context.Context, client ACMAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var acmArns []*string

	// By default, ListCertificates only returns RSA_1024 and RSA_2048 certificates.
	// Explicitly include all key types to ensure we find all certificates.
	input := &acm.ListCertificatesInput{
		Includes: &types.Filters{
			KeyTypes: []types.KeyAlgorithm{
				types.KeyAlgorithmRsa1024,
				types.KeyAlgorithmRsa2048,
				types.KeyAlgorithmRsa3072,
				types.KeyAlgorithmRsa4096,
				types.KeyAlgorithmEcPrime256v1,
				types.KeyAlgorithmEcSecp384r1,
				types.KeyAlgorithmEcSecp521r1,
			},
		},
	}
	paginator := acm.NewListCertificatesPaginator(client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cert := range page.CertificateSummaryList {
			if shouldIncludeACMCertificate(cert, cfg) {
				acmArns = append(acmArns, cert.CertificateArn)
			}
		}
	}

	return acmArns, nil
}

// shouldIncludeACMCertificate determines if an ACM certificate should be included based on config filters.
func shouldIncludeACMCertificate(cert types.CertificateSummary, cfg config.ResourceType) bool {
	if aws.ToBool(cert.InUse) {
		logging.Debugf("ACM %s is in use, skipping", aws.ToString(cert.CertificateArn))
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: cert.DomainName,
		Time: cert.CreatedAt,
	})
}

// deleteACMCertificate deletes a single ACM certificate.
func deleteACMCertificate(ctx context.Context, client ACMAPI, arn *string) error {
	_, err := client.DeleteCertificate(ctx, &acm.DeleteCertificateInput{
		CertificateArn: arn,
	})
	if err != nil {
		return fmt.Errorf("failed to delete ACM certificate %s: %w", aws.ToString(arn), err)
	}
	return nil
}
