package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

// mockedCloudMapNamespacesAPI is a mock implementation of CloudMapNamespacesAPI for testing.
// It returns predefined responses for API calls, allowing tests to run without AWS credentials.
type mockedCloudMapNamespacesAPI struct {
	CloudMapNamespacesAPI
	ListNamespacesOutput  servicediscovery.ListNamespacesOutput
	DeleteNamespaceOutput servicediscovery.DeleteNamespaceOutput
	GetNamespaceOutput    servicediscovery.GetNamespaceOutput
	ListServicesOutput    servicediscovery.ListServicesOutput
}

func (m mockedCloudMapNamespacesAPI) ListNamespaces(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
	return &m.ListNamespacesOutput, nil
}

func (m mockedCloudMapNamespacesAPI) DeleteNamespace(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error) {
	return &m.DeleteNamespaceOutput, nil
}

func (m mockedCloudMapNamespacesAPI) GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error) {
	return &m.GetNamespaceOutput, nil
}

func (m mockedCloudMapNamespacesAPI) ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

// TestCloudMapNamespaces_GetAll verifies that namespace filtering works correctly.
// It tests empty filters, name exclusion, and time-based exclusion.
func TestCloudMapNamespaces_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testNamespace1 := "ns-123456789"
	testNamespace2 := "ns-987654321"
	testNamespace1Name := "test-namespace-1"
	testNamespace2Name := "test-namespace-2"

	cns := CloudMapNamespaces{
		Client: mockedCloudMapNamespacesAPI{
			ListNamespacesOutput: servicediscovery.ListNamespacesOutput{
				Namespaces: []types.NamespaceSummary{
					{
						Id:         aws.String(testNamespace1),
						Name:       aws.String(testNamespace1Name),
						CreateDate: aws.Time(now.Add(-1 * time.Hour)),
					},
					{
						Id:         aws.String(testNamespace2),
						Name:       aws.String(testNamespace2Name),
						CreateDate: aws.Time(now),
					},
				},
			},
		},
	}
	cns.BaseAwsResource.Init(aws.Config{})

	// Define test cases for different filter scenarios
	tests := map[string]struct {
		configObj config.Config
		expected  []string
	}{
		"emptyFilter": { // Should return all namespaces when no filters are applied
			configObj: config.Config{
				CloudMapNamespace: config.ResourceType{},
			},
			expected: []string{testNamespace1, testNamespace2},
		},
		"nameExclusionFilter": { // Should exclude namespaces matching the regex pattern
			configObj: config.Config{
				CloudMapNamespace: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile("test-namespace-1"),
						}},
					},
				},
			},
			expected: []string{testNamespace2},
		},
		"timeAfterExclusionFilter": { // Should exclude namespaces created after the specified time
			configObj: config.Config{
				CloudMapNamespace: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now.Add(-30 * time.Minute)),
					},
				},
			},
			expected: []string{testNamespace1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := cns.getAll(context.Background(), tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

// TestCloudMapNamespaces_NukeAll verifies that the namespace deletion process works correctly.
// It tests that namespaces can be deleted when they have no services.
func TestCloudMapNamespaces_NukeAll(t *testing.T) {
	t.Parallel()

	cns := CloudMapNamespaces{
		Client: mockedCloudMapNamespacesAPI{
			ListServicesOutput: servicediscovery.ListServicesOutput{
				Services: []types.ServiceSummary{},
			},
			GetNamespaceOutput: servicediscovery.GetNamespaceOutput{
				Namespace: &types.Namespace{
					Id:   aws.String("ns-123456789"),
					Name: aws.String("test-namespace"),
				},
			},
			DeleteNamespaceOutput: servicediscovery.DeleteNamespaceOutput{
				OperationId: aws.String("operation-123"),
			},
		},
	}
	cns.BaseAwsResource.Init(aws.Config{})

	err := cns.nukeAll([]*string{aws.String("ns-123456789")})
	require.NoError(t, err)
}