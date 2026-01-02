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
	UpdateCalled                       bool
}

func (m *mockACMPCAClient) ListCertificateAuthorities(ctx context.Context, params *acmpca.ListCertificateAuthoritiesInput, optFns ...func(*acmpca.Options)) (*acmpca.ListCertificateAuthoritiesOutput, error) {
	return &m.ListCertificateAuthoritiesOutput, nil
}

func (m *mockACMPCAClient) DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error) {
	return &m.DescribeCertificateAuthorityOutput, nil
}

func (m *mockACMPCAClient) UpdateCertificateAuthority(ctx context.Context, params *acmpca.UpdateCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.UpdateCertificateAuthorityOutput, error) {
	m.UpdateCalled = true
	return &m.UpdateCertificateAuthorityOutput, nil
}

func (m *mockACMPCAClient) DeleteCertificateAuthority(ctx context.Context, params *acmpca.DeleteCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DeleteCertificateAuthorityOutput, error) {
	return &m.DeleteCertificateAuthorityOutput, nil
}

func TestListACMPCA(t *testing.T) {
	t.Parallel()

	testArn1 := "arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/test-ca-1"
	testArn2 := "arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/test-ca-2"
	now := time.Now()
	earlier := now.Add(-time.Hour)

	tests := map[string]struct {
		cas      []types.CertificateAuthority
		cfg      config.ResourceType
		expected []string
	}{
		"emptyFilter": {
			cas: []types.CertificateAuthority{
				{CreatedAt: &now, Arn: aws.String(testArn1)},
				{CreatedAt: &now, Arn: aws.String(testArn2)},
			},
			cfg:      config.ResourceType{},
			expected: []string{testArn1, testArn2},
		},
		"timeAfterExclusionFilter": {
			cas: []types.CertificateAuthority{
				{CreatedAt: &now, Arn: aws.String(testArn1)},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1))},
			},
			expected: nil,
		},
		"excludesDeletedStatus": {
			cas: []types.CertificateAuthority{
				{CreatedAt: &now, Arn: aws.String(testArn1), Status: types.CertificateAuthorityStatusDeleted},
			},
			cfg:      config.ResourceType{},
			expected: nil,
		},
		"usesLastStateChangeAtOverCreatedAt": {
			cas: []types.CertificateAuthority{
				{CreatedAt: &earlier, LastStateChangeAt: &now, Arn: aws.String(testArn1)},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1))},
			},
			expected: nil, // Excluded because lastStateChangeAt (now) is after the filter
		},
		"usesCreatedAtWhenLastStateChangeAtIsNil": {
			cas: []types.CertificateAuthority{
				{CreatedAt: &earlier, LastStateChangeAt: nil, Arn: aws.String(testArn1)},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1))},
			},
			expected: []string{testArn1}, // Included because createdAt (earlier) is before the filter
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

func TestDeleteACMPCA(t *testing.T) {
	t.Parallel()

	testArn := "arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/test-ca"
	now := time.Now()

	tests := map[string]struct {
		status             types.CertificateAuthorityStatus
		expectUpdateCalled bool
	}{
		"activeCA_requiresDisable": {
			status:             types.CertificateAuthorityStatusActive,
			expectUpdateCalled: true,
		},
		"disabledCA_noUpdateNeeded": {
			status:             types.CertificateAuthorityStatusDisabled,
			expectUpdateCalled: false,
		},
		"pendingCertificateCA_noUpdateNeeded": {
			status:             types.CertificateAuthorityStatusPendingCertificate,
			expectUpdateCalled: false,
		},
		"creatingCA_noUpdateNeeded": {
			status:             types.CertificateAuthorityStatusCreating,
			expectUpdateCalled: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockACMPCAClient{
				DescribeCertificateAuthorityOutput: acmpca.DescribeCertificateAuthorityOutput{
					CertificateAuthority: &types.CertificateAuthority{
						Status:    tc.status,
						CreatedAt: &now,
						Arn:       aws.String(testArn),
					},
				},
			}

			err := deleteACMPCA(context.Background(), mock, aws.String(testArn))
			require.NoError(t, err)
			require.Equal(t, tc.expectUpdateCalled, mock.UpdateCalled, "UpdateCertificateAuthority call mismatch")
		})
	}
}
