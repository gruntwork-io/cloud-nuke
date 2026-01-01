package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockACMClient struct {
	ListCertificatesOutput  acm.ListCertificatesOutput
	DeleteCertificateOutput acm.DeleteCertificateOutput
}

func (m *mockACMClient) ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	return &m.ListCertificatesOutput, nil
}

func (m *mockACMClient) DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
	return &m.DeleteCertificateOutput, nil
}

func TestListACMCertificates(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testArn := "arn:aws:acm:us-east-1:123456789012:certificate/test-cert"
	testDomain := "test.example.com"

	tests := []struct {
		name     string
		certs    []types.CertificateSummary
		cfg      config.ResourceType
		expected []string
	}{
		{
			name: "returns all certificates without filters",
			certs: []types.CertificateSummary{
				{CertificateArn: aws.String(testArn), DomainName: aws.String(testDomain), CreatedAt: aws.Time(now)},
			},
			cfg:      config.ResourceType{},
			expected: []string{testArn},
		},
		{
			name: "filters by name regex",
			certs: []types.CertificateSummary{
				{CertificateArn: aws.String(testArn), DomainName: aws.String(testDomain), CreatedAt: aws.Time(now)},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test.*")}},
				},
			},
			expected: nil,
		},
		{
			name: "excludes in-use certificates",
			certs: []types.CertificateSummary{
				{CertificateArn: aws.String(testArn), DomainName: aws.String(testDomain), InUse: aws.Bool(true)},
			},
			cfg:      config.ResourceType{},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockACMClient{
				ListCertificatesOutput: acm.ListCertificatesOutput{
					CertificateSummaryList: tc.certs,
				},
			}

			arns, err := listACMCertificates(context.Background(), mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestDeleteACMCertificate(t *testing.T) {
	t.Parallel()

	mock := &mockACMClient{}
	err := deleteACMCertificate(context.Background(), mock, aws.String("test-arn"))
	require.NoError(t, err)
}
