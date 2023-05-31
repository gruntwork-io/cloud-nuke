package aws

import (
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func TestShouldIncludeACM(t *testing.T) {
	certSummary := func(domainName string, createdAt time.Time) *acm.CertificateSummary {
		return &acm.CertificateSummary{
			CertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"),
			DomainName:     aws.String(domainName),
			CreatedAt:      aws.Time(createdAt),
		}
	}

	tests := map[string]struct {
		acm          *acm.CertificateSummary
		excludeAfter time.Time
		configObj    config.Config
		expected     bool
	}{
		"include on domain name": {
			acm:          certSummary("example.com", time.Now()),
			excludeAfter: time.Now(),
			configObj: config.Config{
				ACM: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{
							{
								RE: *regexp.MustCompile("example.com"),
							},
						},
					},
				},
			},
			expected: true,
		},
		"exclude on domain name": {
			acm:          certSummary("example.com", time.Now()),
			excludeAfter: time.Now(),
			configObj: config.Config{
				ACM: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{
							{
								RE: *regexp.MustCompile("example.com"),
							},
						},
					},
				},
			},
			expected: false,
		},
		"include on domain name, exclude on created at": {
			acm:          certSummary("example.com", time.Now()),
			excludeAfter: time.Now().Add(-1 * time.Hour),
			configObj: config.Config{
				ACM: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{
							{
								RE: *regexp.MustCompile("example.com"),
							},
						},
					},
				},
			},
			expected: false,
		},
		"nil cert summary": {
			acm:          nil,
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"nil created at": {
			acm:          certSummary("example.com", time.Time{}),
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := shouldIncludeACM(test.acm, test.excludeAfter, test.configObj)

			if actual != test.expected {
				t.Errorf("Expected %t, but got %t", test.expected, actual)
			}
		})
	}
}
