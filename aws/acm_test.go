package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func createHTTPListenerFromListCertificatesOutput(t *testing.T, listCertificatesOutput *acm.ListCertificatesOutput, responseCode int) *httptest.Server {
	response, err := json.Marshal(listCertificatesOutput)
	if err != nil {
		t.Fatalf("Could not marshal certificate summary: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseCode)
		w.Write(response)
	})

	return httptest.NewServer(mux)
}

func createTestSession(t *testing.T, url string) *session.Session {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("no-region"),
		Credentials: credentials.NewStaticCredentials(
			"test",
			"test",
			"test",
		),
		DisableSSL: aws.Bool(true),
		Endpoint:   aws.String(url),
	})
	if err != nil {
		t.Fatalf("Could not create test session: %s", err)
	}

	return session
}

func TestGetAllACMs(t *testing.T) {
	createListCertitificatesOutput := func(certArns []string, domainNames []string, createdTimes []time.Time) *acm.ListCertificatesOutput {
		certificates := []*acm.CertificateSummary{}
		for i := range certArns {
			certificates = append(certificates, &acm.CertificateSummary{
				CertificateArn: aws.String(certArns[i]),
				DomainName:     aws.String(domainNames[i]),
				CreatedAt:      aws.Time(createdTimes[i]),
			})
		}

		return &acm.ListCertificatesOutput{
			CertificateSummaryList: certificates,
		}
	}

	tests := map[string]struct {
		listCertificatesOutput *acm.ListCertificatesOutput
		excludeAfter           time.Time
		configObj              config.Config
		expected               []string
	}{
		"no acms": {
			listCertificatesOutput: &acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{},
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     []string{},
		},
		"single acm": {
			listCertificatesOutput: createListCertitificatesOutput(
				[]string{"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"},
				[]string{"example.com"},
				[]time.Time{time.Now()},
			),
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected: []string{
				"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
			},
		},
		"multiple acms": {
			listCertificatesOutput: createListCertitificatesOutput(
				[]string{
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789013",
				},
				[]string{
					"example.com",
					"example.org",
				},
				[]time.Time{
					time.Now(),
					time.Now(),
				},
			),
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected: []string{
				"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
				"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789013",
			},
		},
		"exclude after": {
			listCertificatesOutput: createListCertitificatesOutput(
				[]string{
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789013",
				},
				[]string{
					"example.com",
					"example.org",
				},
				[]time.Time{
					time.Now(),
					time.Now(),
				},
			),
			excludeAfter: time.Now().Add(-1 * time.Hour),
			configObj:    config.Config{},
			expected:     []string{},
		},
		"include on domain name": {
			listCertificatesOutput: createListCertitificatesOutput(
				[]string{
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789013",
				},
				[]string{
					"example.com",
					"example.org",
				},
				[]time.Time{
					time.Now(),
					time.Now(),
				},
			),
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
			expected: []string{
				"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
			},
		},
		"exclude on domain name": {
			listCertificatesOutput: createListCertitificatesOutput(
				[]string{
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
					"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789013",
				},
				[]string{
					"example.com",
					"example.org",
				},
				[]time.Time{
					time.Now(),
					time.Now(),
				},
			),
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
			expected: []string{
				"arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789013",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			httpServer := createHTTPListenerFromListCertificatesOutput(t, test.listCertificatesOutput, http.StatusOK)
			defer httpServer.Close()

			session := createTestSession(t, httpServer.URL)

			actual, err := getAllACMs(session, test.excludeAfter, test.configObj)
			if err != nil {
				t.Errorf("Expected no error, but got %s", err)
			}

			if len(actual) != len(test.expected) {
				t.Errorf("Expected %d, but got %d", len(test.expected), len(actual))
			}
		})
	}
}

func TestGetAllACMsError(t *testing.T) {
	httpServer := createHTTPListenerFromListCertificatesOutput(t, &acm.ListCertificatesOutput{}, http.StatusInternalServerError)
	defer httpServer.Close()

	session := createTestSession(t, httpServer.URL)

	_, err := getAllACMs(session, time.Now(), config.Config{})
	if err == nil {
		t.Errorf("Expected error, but got none")
	}
}

func TestShouldIncludeACM(t *testing.T) {
	certSummary := func(domainName string, createdAt time.Time, inUse bool) *acm.CertificateSummary {
		return &acm.CertificateSummary{
			CertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"),
			DomainName:     aws.String(domainName),
			CreatedAt:      aws.Time(createdAt),
			InUse:          aws.Bool(inUse),
		}
	}

	tests := map[string]struct {
		acm          *acm.CertificateSummary
		excludeAfter time.Time
		configObj    config.Config
		expected     bool
	}{
		"include on domain name": {
			acm:          certSummary("example.com", time.Now(), false),
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
			acm:          certSummary("example.com", time.Now(), false),
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
			acm:          certSummary("example.com", time.Now(), false),
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
		"exclude on domain name, include on created at": {
			acm:          certSummary("example.com", time.Now(), false),
			excludeAfter: time.Now().Add(-1 * time.Hour),
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
		"in use": {
			acm:          certSummary("example.com", time.Now(), true),
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"not in use": {
			acm:          certSummary("example.com", time.Now(), false),
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     true,
		},
		"nil cert summary": {
			acm:          nil,
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"nil created at": {
			acm:          certSummary("example.com", time.Time{}, false),
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
