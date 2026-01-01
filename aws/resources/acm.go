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
	paginator := acm.NewListCertificatesPaginator(client, &acm.ListCertificatesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cert := range page.CertificateSummaryList {
			logging.Debugf("Found ACM %s with domain name %s", *cert.CertificateArn, *cert.DomainName)
			if shouldIncludeACMCertificate(cert, cfg) {
				logging.Debugf("Including ACM %s", *cert.CertificateArn)
				acmArns = append(acmArns, cert.CertificateArn)
			} else {
				logging.Debugf("Skipping ACM %s", *cert.CertificateArn)
			}
		}
	}

	return acmArns, nil
}

// shouldIncludeACMCertificate determines if an ACM certificate should be included based on config filters.
func shouldIncludeACMCertificate(cert types.CertificateSummary, cfg config.ResourceType) bool {
	if cert.InUse != nil && *cert.InUse {
		logging.Debugf("ACM %s is in use", *cert.CertificateArn)
		return false
	}

	shouldInclude := cfg.ShouldInclude(config.ResourceValue{
		Name: cert.DomainName,
		Time: cert.CreatedAt,
	})
	logging.Debugf("shouldInclude result for ACM: %s w/ domain name: %s, time: %s, and config: %+v",
		aws.ToString(cert.CertificateArn), aws.ToString(cert.DomainName), cert.CreatedAt, cfg)
	return shouldInclude
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
