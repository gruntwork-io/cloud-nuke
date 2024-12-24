package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ACMPCAServiceAPI interface {
	DeleteCertificateAuthority(ctx context.Context, params *acmpca.DeleteCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DeleteCertificateAuthorityOutput, error)
	DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error)
	ListCertificateAuthorities(ctx context.Context, params *acmpca.ListCertificateAuthoritiesInput, optFns ...func(*acmpca.Options)) (*acmpca.ListCertificateAuthoritiesOutput, error)
	UpdateCertificateAuthority(ctx context.Context, params *acmpca.UpdateCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.UpdateCertificateAuthorityOutput, error)
}

// ACMPCA - represents all ACMPA
type ACMPCA struct {
	BaseAwsResource
	Client ACMPCAServiceAPI
	Region string
	ARNs   []string
}

func (ap *ACMPCA) Init(cfg aws.Config) {
	ap.Client = acmpca.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ap *ACMPCA) ResourceName() string {
	return "acmpca"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ap *ACMPCA) ResourceIdentifiers() []string {
	return ap.ARNs
}

func (ap *ACMPCA) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}
func (ap *ACMPCA) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ACMPCA
}

func (ap *ACMPCA) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ap.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ap.ARNs = aws.ToStringSlice(identifiers)
	return ap.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (ap *ACMPCA) Nuke(arns []string) error {
	if err := ap.nukeAll(aws.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
