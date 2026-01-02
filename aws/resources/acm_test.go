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
	testArn1 := "arn:aws:acm:us-east-1:123456789012:certificate/test-cert-1"
	testArn2 := "arn:aws:acm:us-east-1:123456789012:certificate/test-cert-2"
	testDomain1 := "test.example.com"
	testDomain2 := "other.example.com"

	mock := &mockACMClient{
		ListCertificatesOutput: acm.ListCertificatesOutput{
			CertificateSummaryList: []types.CertificateSummary{
				{CertificateArn: aws.String(testArn1), DomainName: aws.String(testDomain1), CreatedAt: aws.Time(now)},
				{CertificateArn: aws.String(testArn2), DomainName: aws.String(testDomain2), CreatedAt: aws.Time(now.Add(time.Hour))},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test.*")}},
				},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testArn1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			arns, err := listACMCertificates(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestShouldIncludeACMCertificate(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		cert     types.CertificateSummary
		cfg      config.ResourceType
		expected bool
	}{
		"includeWhenNotInUse": {
			cert: types.CertificateSummary{
				CertificateArn: aws.String("test-arn"),
				DomainName:     aws.String("test.example.com"),
				InUse:          aws.Bool(false),
				CreatedAt:      aws.Time(now),
			},
			cfg:      config.ResourceType{},
			expected: true,
		},
		"excludeWhenInUse": {
			cert: types.CertificateSummary{
				CertificateArn: aws.String("test-arn"),
				DomainName:     aws.String("test.example.com"),
				InUse:          aws.Bool(true),
				CreatedAt:      aws.Time(now),
			},
			cfg:      config.ResourceType{},
			expected: false,
		},
		"excludeWhenInUseIsNil": {
			cert: types.CertificateSummary{
				CertificateArn: aws.String("test-arn"),
				DomainName:     aws.String("test.example.com"),
				InUse:          nil,
				CreatedAt:      aws.Time(now),
			},
			cfg:      config.ResourceType{},
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := shouldIncludeACMCertificate(tc.cert, tc.cfg)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestDeleteACMCertificate(t *testing.T) {
	t.Parallel()

	mock := &mockACMClient{}
	err := deleteACMCertificate(context.Background(), mock, aws.String("test-arn"))
	require.NoError(t, err)
}
