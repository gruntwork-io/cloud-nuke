package resources

import (
	"context"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/acmpca/acmpcaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedACMPCA struct {
	acmpcaiface.ACMPCAAPI
	ListCertificateAuthoritiesOutput   acmpca.ListCertificateAuthoritiesOutput
	DescribeCertificateAuthorityOutput acmpca.DescribeCertificateAuthorityOutput
	UpdateCertificateAuthorityOutput   acmpca.UpdateCertificateAuthorityOutput
	DeleteCertificateAuthorityOutput   acmpca.DeleteCertificateAuthorityOutput
}

func (m mockedACMPCA) ListCertificateAuthoritiesPages(
	input *acmpca.ListCertificateAuthoritiesInput, fn func(*acmpca.ListCertificateAuthoritiesOutput, bool) bool) error {
	// Only need to return mocked response output
	fn(&m.ListCertificateAuthoritiesOutput, true)
	return nil
}

func (m mockedACMPCA) DescribeCertificateAuthority(
	input *acmpca.DescribeCertificateAuthorityInput) (*acmpca.DescribeCertificateAuthorityOutput, error) {
	return &m.DescribeCertificateAuthorityOutput, nil
}

func (m mockedACMPCA) UpdateCertificateAuthority(
	input *acmpca.UpdateCertificateAuthorityInput) (*acmpca.UpdateCertificateAuthorityOutput, error) {
	return &m.UpdateCertificateAuthorityOutput, nil
}

func (m mockedACMPCA) DeleteCertificateAuthority(
	input *acmpca.DeleteCertificateAuthorityInput) (*acmpca.DeleteCertificateAuthorityOutput, error) {
	return &m.DeleteCertificateAuthorityOutput, nil
}

func TestAcmPcaGetAll(t *testing.T) {

	t.Parallel()

	testArn := "test-arn"
	now := time.Now()
	acmPca := ACMPCA{
		Client: mockedACMPCA{
			ListCertificateAuthoritiesOutput: acmpca.ListCertificateAuthoritiesOutput{
				CertificateAuthorities: []*acmpca.CertificateAuthority{
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
	require.Contains(t, awsgo.StringValueSlice(arns), testArn)

	// with exclude after filter
	arns, err = acmPca.getAll(context.Background(), config.Config{
		ACMPCA: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: awsgo.Time(now.Add(-1))}},
	})
	require.NoError(t, err)
	require.NotContains(t, awsgo.StringValueSlice(arns), testArn)
}

func TestAcmPcaNukeAll_DisabledCA(t *testing.T) {

	t.Parallel()

	testArn := "test-arn"
	now := time.Now()
	acmPca := ACMPCA{
		Client: mockedACMPCA{
			DescribeCertificateAuthorityOutput: acmpca.DescribeCertificateAuthorityOutput{
				CertificateAuthority: &acmpca.CertificateAuthority{
					Status:    awsgo.String(acmpca.CertificateAuthorityStatusDisabled),
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
				CertificateAuthority: &acmpca.CertificateAuthority{
					Status:    awsgo.String(acmpca.CertificateAuthorityStatusActive),
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
