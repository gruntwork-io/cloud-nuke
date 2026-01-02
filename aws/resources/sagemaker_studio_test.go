package resources

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockSageMakerStudioClient struct {
	mu sync.Mutex

	// Domain operations
	ListDomainsOutput    sagemaker.ListDomainsOutput
	DescribeDomainOutput sagemaker.DescribeDomainOutput
	ListDomainsError     error

	// UserProfile operations
	ListUserProfilesOutput sagemaker.ListUserProfilesOutput
	DeleteUserProfileError error

	// App operations
	ListAppsOutput sagemaker.ListAppsOutput
	DeleteAppError error

	// Space operations
	ListSpacesOutput sagemaker.ListSpacesOutput
	DeleteSpaceError error

	// MLflow tracking server operations
	ListMlflowTrackingServersOutput sagemaker.ListMlflowTrackingServersOutput

	// Track deleted resources
	deletedApps     map[string]bool
	deletedProfiles map[string]bool
	deletedDomains  map[string]bool
	deletedServers  map[string]bool
	deletedSpaces   map[string]bool
}

func newMockSageMakerStudioClient() *mockSageMakerStudioClient {
	return &mockSageMakerStudioClient{
		deletedApps:     make(map[string]bool),
		deletedProfiles: make(map[string]bool),
		deletedDomains:  make(map[string]bool),
		deletedServers:  make(map[string]bool),
		deletedSpaces:   make(map[string]bool),
	}
}

func (m *mockSageMakerStudioClient) ListDomains(ctx context.Context, params *sagemaker.ListDomainsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListDomainsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ListDomainsError != nil {
		return nil, m.ListDomainsError
	}

	var activeDomains []types.DomainDetails
	for _, domain := range m.ListDomainsOutput.Domains {
		if !m.deletedDomains[*domain.DomainId] {
			activeDomains = append(activeDomains, domain)
		}
	}
	return &sagemaker.ListDomainsOutput{Domains: activeDomains}, nil
}

func (m *mockSageMakerStudioClient) DescribeDomain(ctx context.Context, params *sagemaker.DescribeDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeDomainOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deletedDomains[*params.DomainId] {
		return nil, &smithy.GenericAPIError{Code: "ResourceNotFound", Message: "Domain not found"}
	}
	return &m.DescribeDomainOutput, nil
}

func (m *mockSageMakerStudioClient) DeleteDomain(ctx context.Context, params *sagemaker.DeleteDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteDomainOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedDomains[*params.DomainId] = true
	return &sagemaker.DeleteDomainOutput{}, nil
}

func (m *mockSageMakerStudioClient) ListUserProfiles(ctx context.Context, params *sagemaker.ListUserProfilesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListUserProfilesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var activeProfiles []types.UserProfileDetails
	for _, profile := range m.ListUserProfilesOutput.UserProfiles {
		key := aws.ToString(profile.DomainId) + "/" + aws.ToString(profile.UserProfileName)
		if !m.deletedProfiles[key] {
			activeProfiles = append(activeProfiles, profile)
		}
	}
	return &sagemaker.ListUserProfilesOutput{UserProfiles: activeProfiles}, nil
}

func (m *mockSageMakerStudioClient) DeleteUserProfile(ctx context.Context, params *sagemaker.DeleteUserProfileInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteUserProfileOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteUserProfileError != nil {
		return nil, m.DeleteUserProfileError
	}

	key := aws.ToString(params.DomainId) + "/" + aws.ToString(params.UserProfileName)
	m.deletedProfiles[key] = true
	return &sagemaker.DeleteUserProfileOutput{}, nil
}

func (m *mockSageMakerStudioClient) ListApps(ctx context.Context, params *sagemaker.ListAppsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListAppsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var activeApps []types.AppDetails
	for _, app := range m.ListAppsOutput.Apps {
		key := aws.ToString(app.DomainId) + "/" + aws.ToString(app.UserProfileName) + "/" + aws.ToString(app.AppName)
		if m.deletedApps[key] {
			app.Status = types.AppStatusDeleted
		}
		activeApps = append(activeApps, app)
	}
	return &sagemaker.ListAppsOutput{Apps: activeApps}, nil
}

func (m *mockSageMakerStudioClient) DeleteApp(ctx context.Context, params *sagemaker.DeleteAppInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteAppOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteAppError != nil {
		return nil, m.DeleteAppError
	}

	key := aws.ToString(params.DomainId) + "/"
	if params.UserProfileName != nil {
		key += aws.ToString(params.UserProfileName)
	} else if params.SpaceName != nil {
		key += aws.ToString(params.SpaceName)
	}
	key += "/" + aws.ToString(params.AppName)
	m.deletedApps[key] = true
	return &sagemaker.DeleteAppOutput{}, nil
}

func (m *mockSageMakerStudioClient) ListMlflowTrackingServers(ctx context.Context, params *sagemaker.ListMlflowTrackingServersInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListMlflowTrackingServersOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var activeServers []types.TrackingServerSummary
	for _, server := range m.ListMlflowTrackingServersOutput.TrackingServerSummaries {
		if !m.deletedServers[aws.ToString(server.TrackingServerName)] {
			activeServers = append(activeServers, server)
		}
	}
	return &sagemaker.ListMlflowTrackingServersOutput{TrackingServerSummaries: activeServers}, nil
}

func (m *mockSageMakerStudioClient) DeleteMlflowTrackingServer(ctx context.Context, params *sagemaker.DeleteMlflowTrackingServerInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteMlflowTrackingServerOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedServers[aws.ToString(params.TrackingServerName)] = true
	return &sagemaker.DeleteMlflowTrackingServerOutput{}, nil
}

func (m *mockSageMakerStudioClient) ListSpaces(ctx context.Context, params *sagemaker.ListSpacesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListSpacesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var activeSpaces []types.SpaceDetails
	for _, space := range m.ListSpacesOutput.Spaces {
		key := aws.ToString(space.DomainId) + "/" + aws.ToString(space.SpaceName)
		if !m.deletedSpaces[key] {
			activeSpaces = append(activeSpaces, space)
		}
	}
	return &sagemaker.ListSpacesOutput{Spaces: activeSpaces}, nil
}

func (m *mockSageMakerStudioClient) DeleteSpace(ctx context.Context, params *sagemaker.DeleteSpaceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteSpaceOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteSpaceError != nil {
		return nil, m.DeleteSpaceError
	}

	key := aws.ToString(params.DomainId) + "/" + aws.ToString(params.SpaceName)
	m.deletedSpaces[key] = true
	return &sagemaker.DeleteSpaceOutput{}, nil
}

func TestListSageMakerDomains(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		mock        *mockSageMakerStudioClient
		expected    []string
		expectError bool
	}{
		"singleDomain": {
			mock: &mockSageMakerStudioClient{
				ListDomainsOutput: sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{
						{DomainId: aws.String("domain-1"), DomainName: aws.String("test-domain-1"), CreationTime: &now},
					},
				},
			},
			expected:    []string{"domain-1"},
			expectError: false,
		},
		"multipleDomains": {
			mock: &mockSageMakerStudioClient{
				ListDomainsOutput: sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{
						{DomainId: aws.String("domain-1"), DomainName: aws.String("test-domain-1"), CreationTime: &now},
						{DomainId: aws.String("domain-2"), DomainName: aws.String("test-domain-2"), CreationTime: &now},
					},
				},
			},
			expected:    []string{"domain-1", "domain-2"},
			expectError: false,
		},
		"emptyList": {
			mock:        &mockSageMakerStudioClient{ListDomainsOutput: sagemaker.ListDomainsOutput{}},
			expected:    []string{},
			expectError: false,
		},
		"apiError": {
			mock:        &mockSageMakerStudioClient{ListDomainsError: fmt.Errorf("AWS API error")},
			expected:    nil,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listSageMakerDomains(context.Background(), tc.mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, aws.ToStringSlice(ids))
			}
		})
	}
}

func TestNukeSageMakerDomains(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mock            *mockSageMakerStudioClient
		identifiers     []string
		expectedSuccess int
		expectedFailure int
	}{
		"successWithAppsAndProfiles": {
			mock: func() *mockSageMakerStudioClient {
				m := newMockSageMakerStudioClient()
				m.ListAppsOutput = sagemaker.ListAppsOutput{
					Apps: []types.AppDetails{
						{AppName: aws.String("app-1"), AppType: types.AppTypeJupyterServer, DomainId: aws.String("domain-1"), UserProfileName: aws.String("user-1"), Status: types.AppStatusInService},
					},
				}
				m.ListUserProfilesOutput = sagemaker.ListUserProfilesOutput{
					UserProfiles: []types.UserProfileDetails{
						{DomainId: aws.String("domain-1"), UserProfileName: aws.String("user-1")},
					},
				}
				return m
			}(),
			identifiers:     []string{"domain-1"},
			expectedSuccess: 1,
			expectedFailure: 0,
		},
		"emptyIdentifiers": {
			mock:            newMockSageMakerStudioClient(),
			identifiers:     []string{},
			expectedSuccess: 0,
			expectedFailure: 0,
		},
		"appDeleteError": {
			mock: func() *mockSageMakerStudioClient {
				m := newMockSageMakerStudioClient()
				m.ListAppsOutput = sagemaker.ListAppsOutput{
					Apps: []types.AppDetails{
						{AppName: aws.String("app-1"), AppType: types.AppTypeJupyterServer, DomainId: aws.String("domain-1"), UserProfileName: aws.String("user-1"), Status: types.AppStatusInService},
					},
				}
				m.DeleteAppError = fmt.Errorf("failed to delete app")
				return m
			}(),
			identifiers:     []string{"domain-1"},
			expectedSuccess: 0,
			expectedFailure: 1,
		},
		"userProfileDeleteError": {
			mock: func() *mockSageMakerStudioClient {
				m := newMockSageMakerStudioClient()
				m.ListUserProfilesOutput = sagemaker.ListUserProfilesOutput{
					UserProfiles: []types.UserProfileDetails{
						{DomainId: aws.String("domain-1"), UserProfileName: aws.String("user-1")},
					},
				}
				m.DeleteUserProfileError = fmt.Errorf("failed to delete user profile")
				return m
			}(),
			identifiers:     []string{"domain-1"},
			expectedSuccess: 0,
			expectedFailure: 1,
		},
		"spaceDeleteError": {
			mock: func() *mockSageMakerStudioClient {
				m := newMockSageMakerStudioClient()
				m.ListSpacesOutput = sagemaker.ListSpacesOutput{
					Spaces: []types.SpaceDetails{
						{SpaceName: aws.String("space-1"), DomainId: aws.String("domain-1"), Status: types.SpaceStatusInService},
					},
				}
				m.DeleteSpaceError = fmt.Errorf("failed to delete space")
				return m
			}(),
			identifiers:     []string{"domain-1"},
			expectedSuccess: 0,
			expectedFailure: 1,
		},
		"withMlflowServer": {
			mock: func() *mockSageMakerStudioClient {
				m := newMockSageMakerStudioClient()
				m.ListMlflowTrackingServersOutput = sagemaker.ListMlflowTrackingServersOutput{
					TrackingServerSummaries: []types.TrackingServerSummary{
						{TrackingServerName: aws.String("server-1"), TrackingServerStatus: types.TrackingServerStatusCreated},
					},
				}
				return m
			}(),
			identifiers:     []string{"domain-1"},
			expectedSuccess: 1,
			expectedFailure: 0,
		},
		"withSpaces": {
			mock: func() *mockSageMakerStudioClient {
				m := newMockSageMakerStudioClient()
				m.ListSpacesOutput = sagemaker.ListSpacesOutput{
					Spaces: []types.SpaceDetails{
						{SpaceName: aws.String("space-1"), DomainId: aws.String("domain-1"), Status: types.SpaceStatusInService},
					},
				}
				return m
			}(),
			identifiers:     []string{"domain-1"},
			expectedSuccess: 1,
			expectedFailure: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			identifiers := make([]*string, len(tc.identifiers))
			for i, id := range tc.identifiers {
				identifiers[i] = aws.String(id)
			}

			results := nukeSageMakerDomains(context.Background(), tc.mock, resource.Scope{Region: "us-east-1"}, "sagemaker-studio", identifiers)

			successCount := 0
			failureCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				} else {
					failureCount++
				}
			}

			require.Equal(t, tc.expectedSuccess, successCount, "success count mismatch")
			require.Equal(t, tc.expectedFailure, failureCount, "failure count mismatch")
		})
	}
}
