package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedACMPCA struct {
	ACMPCAServiceAPI
	acmpca.DeleteCertificateAuthorityOutput
	acmpca.DescribeCertificateAuthorityOutput
	acmpca.ListCertificateAuthoritiesOutput
	acmpca.UpdateCertificateAuthorityOutput
}

func (m mockedACMPCA) DeleteCertificateAuthority(ctx context.Context, params *acmpca.DeleteCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DeleteCertificateAuthorityOutput, error) {
	return &m.DeleteCertificateAuthorityOutput, nil
}

func (m mockedACMPCA) DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error) {
	return &m.DescribeCertificateAuthorityOutput, nil
}

func (m mockedACMPCA) ListCertificateAuthorities(ctx context.Context, params *acmpca.ListCertificateAuthoritiesInput, optFns ...func(*acmpca.Options)) (*acmpca.ListCertificateAuthoritiesOutput, error) {
	return &m.ListCertificateAuthoritiesOutput, nil
}

func (m mockedACMPCA) UpdateCertificateAuthority(ctx context.Context, params *acmpca.UpdateCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.UpdateCertificateAuthorityOutput, error) {
	return &m.UpdateCertificateAuthorityOutput, nil
}

func TestAcmPcaGetAll(t *testing.T) {
	t.Parallel()

	testArn := "test-arn"
	now := time.Now()
	acmPca := ACMPCA{
		Client: mockedACMPCA{
			ListCertificateAuthoritiesOutput: acmpca.ListCertificateAuthoritiesOutput{
				CertificateAuthorities: []types.CertificateAuthority{
					{
						CreatedAt: &now,
						Arn:       &testArn,
					},
				},
			},
		},
	}

	// without filters
	arns, err := acmPca.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(arns), testArn)

	// with exclude after filter
	arns, err = acmPca.getAll(context.Background(), config.Config{
		ACMPCA: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-1))}},
	})

	require.NoError(t, err)
	require.NotContains(t, aws.ToStringSlice(arns), testArn)
}

func TestAcmPcaNukeAll_DisabledCA(t *testing.T) {
	t.Parallel()

	testArn := "test-arn"
	now := time.Now()
	acmPca := ACMPCA{
		Client: mockedACMPCA{
			DescribeCertificateAuthorityOutput: acmpca.DescribeCertificateAuthorityOutput{
				CertificateAuthority: &types.CertificateAuthority{
					Status:    types.CertificateAuthorityStatusDisabled,
					CreatedAt: &now,
					Arn:       &testArn,
				},
			},
			DeleteCertificateAuthorityOutput: acmpca.DeleteCertificateAuthorityOutput{},
		},
	}

	err := acmPca.nukeAll([]*string{&testArn})
	require.NoError(t, err)
}

func TestAcmPcaNukeAll_EnabledCA(t *testing.T) {
	t.Parallel()

	testArn := "test-arn"
	now := time.Now()
	acmPca := ACMPCA{
		Client: mockedACMPCA{
			DescribeCertificateAuthorityOutput: acmpca.DescribeCertificateAuthorityOutput{
				CertificateAuthority: &types.CertificateAuthority{
					Status:    types.CertificateAuthorityStatusActive,
					CreatedAt: &now,
					Arn:       &testArn,
				},
			},
			UpdateCertificateAuthorityOutput: acmpca.UpdateCertificateAuthorityOutput{},
			DeleteCertificateAuthorityOutput: acmpca.DeleteCertificateAuthorityOutput{},
		},
	}

	err := acmPca.nukeAll([]*string{&testArn})
	require.NoError(t, err)
}
