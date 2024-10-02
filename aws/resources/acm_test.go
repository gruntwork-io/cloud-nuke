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
	"github.com/stretchr/testify/require"
)

type mockedACM struct {
	ACMServiceAPI
	DeleteCertificateOutput acm.DeleteCertificateOutput
	ListCertificatesOutput  acm.ListCertificatesOutput
}

func (m mockedACM) ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	return &m.ListCertificatesOutput, nil
}

func (m mockedACM) DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
	return &m.DeleteCertificateOutput, nil
}

func TestACMGetAll(t *testing.T) {
	t.Parallel()

	testDomainName := "test-domain-name"
	testArn := "test-arn"
	now := time.Now()
	acmService := ACM{
		Client: mockedACM{
			ListCertificatesOutput: acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{
					{
						DomainName:     &testDomainName,
						CreatedAt:      &now,
						CertificateArn: &testArn,
					},
				},
			},
		},
	}

	// without any filters
	acms, err := acmService.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(acms), testArn)

	// filtering domain names
	acms, err = acmService.getAll(context.Background(), config.Config{
		ACM: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-domain"),
				}}}},
	})
	require.NoError(t, err)
	require.NotContains(t, aws.ToStringSlice(acms), testArn)

	// filtering with time
	acms, err = acmService.getAll(context.Background(), config.Config{
		ACM: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-1)),
			}},
	})
	require.NoError(t, err)
	require.NotContains(t, aws.ToStringSlice(acms), testArn)
}

func TestACMGetAll_FilterInUse(t *testing.T) {
	t.Parallel()

	testDomainName := "test-domain-name"
	testArn := "test-arn"
	acmService := ACM{
		Client: mockedACM{
			ListCertificatesOutput: acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{
					{
						DomainName:     &testDomainName,
						InUse:          aws.Bool(true),
						CertificateArn: &testArn,
					},
				},
			},
		},
	}

	acms, err := acmService.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.NotContains(t, acms, testArn)
}

func TestACMNukeAll(t *testing.T) {
	t.Parallel()

	testDomainName := "test-domain-name"
	acmService := ACM{
		Client: mockedACM{
			DeleteCertificateOutput: acm.DeleteCertificateOutput{},
		},
	}

	err := acmService.nukeAll([]*string{&testDomainName})
	require.NoError(t, err)
}
