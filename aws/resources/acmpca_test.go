package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockACMPCAClient struct {
	ListCertificateAuthoritiesOutput   acmpca.ListCertificateAuthoritiesOutput
	DescribeCertificateAuthorityOutput acmpca.DescribeCertificateAuthorityOutput
	UpdateCertificateAuthorityOutput   acmpca.UpdateCertificateAuthorityOutput
	DeleteCertificateAuthorityOutput   acmpca.DeleteCertificateAuthorityOutput
}

func (m *mockACMPCAClient) ListCertificateAuthorities(ctx context.Context, params *acmpca.ListCertificateAuthoritiesInput, optFns ...func(*acmpca.Options)) (*acmpca.ListCertificateAuthoritiesOutput, error) {
	return &m.ListCertificateAuthoritiesOutput, nil
}

func (m *mockACMPCAClient) DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error) {
	return &m.DescribeCertificateAuthorityOutput, nil
}

func (m *mockACMPCAClient) UpdateCertificateAuthority(ctx context.Context, params *acmpca.UpdateCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.UpdateCertificateAuthorityOutput, error) {
	return &m.UpdateCertificateAuthorityOutput, nil
}

func (m *mockACMPCAClient) DeleteCertificateAuthority(ctx context.Context, params *acmpca.DeleteCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DeleteCertificateAuthorityOutput, error) {
	return &m.DeleteCertificateAuthorityOutput, nil
}

func TestListACMPCA(t *testing.T) {
	t.Parallel()

	testArn := "test-arn"
	now := time.Now()

	tests := []struct {
		name     string
		cas      []types.CertificateAuthority
		cfg      config.ResourceType
		expected []string
	}{
		{
			name: "returns all CAs without filters",
			cas: []types.CertificateAuthority{
				{
					CreatedAt: &now,
					Arn:       &testArn,
				},
			},
			cfg:      config.ResourceType{},
			expected: []string{testArn},
		},
		{
			name: "excludes CAs with exclude after filter",
			cas: []types.CertificateAuthority{
				{
					CreatedAt: &now,
					Arn:       &testArn,
				},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1)),
				},
			},
			expected: nil,
		},
		{
			name: "excludes deleted CAs",
			cas: []types.CertificateAuthority{
				{
					CreatedAt: &now,
					Arn:       &testArn,
					Status:    types.CertificateAuthorityStatusDeleted,
				},
			},
			cfg:      config.ResourceType{},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockACMPCAClient{
				ListCertificateAuthoritiesOutput: acmpca.ListCertificateAuthoritiesOutput{
					CertificateAuthorities: tc.cas,
				},
			}

			arns, err := listACMPCA(context.Background(), mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestDeleteACMPCA_DisabledCA(t *testing.T) {
	t.Parallel()

	testArn := "test-arn"
	now := time.Now()

	mock := &mockACMPCAClient{
		DescribeCertificateAuthorityOutput: acmpca.DescribeCertificateAuthorityOutput{
			CertificateAuthority: &types.CertificateAuthority{
				Status:    types.CertificateAuthorityStatusDisabled,
				CreatedAt: &now,
				Arn:       &testArn,
			},
		},
	}

	err := deleteACMPCA(context.Background(), mock, &testArn)
	require.NoError(t, err)
}

func TestDeleteACMPCA_EnabledCA(t *testing.T) {
	t.Parallel()

	testArn := "test-arn"
	now := time.Now()

	mock := &mockACMPCAClient{
		DescribeCertificateAuthorityOutput: acmpca.DescribeCertificateAuthorityOutput{
			CertificateAuthority: &types.CertificateAuthority{
				Status:    types.CertificateAuthorityStatusActive,
				CreatedAt: &now,
				Arn:       &testArn,
			},
		},
	}

	err := deleteACMPCA(context.Background(), mock, &testArn)
	require.NoError(t, err)
}
