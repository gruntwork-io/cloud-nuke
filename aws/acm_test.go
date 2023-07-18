package aws

import (
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/gruntwork-io/cloud-nuke/config"
)

type mockedACM struct {
	acmiface.ACMAPI
	ListCertificatesOutput  acm.ListCertificatesOutput
	DeleteCertificateOutput acm.DeleteCertificateOutput
}

func (m mockedACM) ListCertificatesPages(
	input *acm.ListCertificatesInput, fn func(*acm.ListCertificatesOutput, bool) bool) error {
	// Only need to return mocked response output
	fn(&m.ListCertificatesOutput, true)
	return nil
}

func (m mockedACM) DeleteCertificate(input *acm.DeleteCertificateInput) (*acm.DeleteCertificateOutput, error) {
	return &m.DeleteCertificateOutput, nil
}

func TestACMGetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testDomainName := "test-domain-name"
	testArn := "test-arn"
	now := time.Now()
	acm := ACM{
		Client: mockedACM{
			ListCertificatesOutput: acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{
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
	acms, err := acm.getAll(config.Config{})
	require.NoError(t, err)
	require.Contains(t, acms, testArn)

	// filtering domain names
	acms, err = acm.getAll(config.Config{
		ACM: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-domain"),
				}}}},
	})
	require.NoError(t, err)
	require.NotContains(t, acms, testArn)

	// filtering with time
	acms, err = acm.getAll(config.Config{
		ACM: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-1)),
			}},
	})
	require.NoError(t, err)
	require.NotContains(t, acms, testArn)
}

func TestACMGetAll_FilterInUse(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testDomainName := "test-domain-name"
	testArn := "test-arn"
	acm := ACM{
		Client: mockedACM{
			ListCertificatesOutput: acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{
					{
						DomainName:     &testDomainName,
						InUse:          aws.Bool(true),
						CertificateArn: &testArn,
					},
				},
			},
		},
	}

	acms, err := acm.getAll(config.Config{})
	require.NoError(t, err)
	require.NotContains(t, acms, testArn)
}

func TestACMNukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testDomainName := "test-domain-name"
	acm := ACM{
		Client: mockedACM{
			DeleteCertificateOutput: acm.DeleteCertificateOutput{},
		},
	}

	err := acm.nukeAll([]*string{&testDomainName})
	require.NoError(t, err)
}
