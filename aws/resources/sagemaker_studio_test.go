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
	"github.com/stretchr/testify/assert"
)

// Mocked SageMaker Studio client for testing
type mockedSageMakerStudio struct {
	SageMakerStudioAPI
	mu sync.Mutex

	// Domain operations
	ListDomainsOutput    sagemaker.ListDomainsOutput
	DescribeDomainOutput sagemaker.DescribeDomainOutput
	DeleteDomainOutput   sagemaker.DeleteDomainOutput
	DescribeDomainError  error
	ListDomainsError     error
	domainStatus         map[string]types.DomainStatus

	// UserProfile operations
	ListUserProfilesOutput    sagemaker.ListUserProfilesOutput
	DeleteUserProfileOutput   sagemaker.DeleteUserProfileOutput
	DescribeUserProfileOutput sagemaker.DescribeUserProfileOutput
	DescribeUserProfileError  error
	DeleteUserProfileError    error

	// App operations
	ListAppsOutput    sagemaker.ListAppsOutput
	DeleteAppOutput   sagemaker.DeleteAppOutput
	DescribeAppOutput sagemaker.DescribeAppOutput
	DescribeAppError  error
	DeleteAppError    error

	// Space operations
	ListSpacesOutput  sagemaker.ListSpacesOutput
	DeleteSpaceOutput sagemaker.DeleteSpaceOutput
	ListSpacesError   error
	DeleteSpaceError  error

	// MLflow tracking server operations
	ListMlflowTrackingServersOutput    sagemaker.ListMlflowTrackingServersOutput
	DeleteMlflowTrackingServerOutput   sagemaker.DeleteMlflowTrackingServerOutput
	DescribeMlflowTrackingServerOutput sagemaker.DescribeMlflowTrackingServerOutput
	ListMlflowTrackingServersError     error

	// Track deleted resources
	deletedApps     map[string]bool
	deletedProfiles map[string]bool
	deletedDomains  map[string]bool
	deletedServers  map[string]bool
	deletedSpaces   map[string]bool
}

func newMockedSageMakerStudio() *mockedSageMakerStudio {
	return &mockedSageMakerStudio{
		deletedApps:     make(map[string]bool),
		deletedProfiles: make(map[string]bool),
		deletedDomains:  make(map[string]bool),
		deletedServers:  make(map[string]bool),
		deletedSpaces:   make(map[string]bool),
		domainStatus:    make(map[string]types.DomainStatus),
	}
}

func (m *mockedSageMakerStudio) ListDomains(ctx context.Context, params *sagemaker.ListDomainsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListDomainsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ListDomainsError != nil {
		return nil, m.ListDomainsError
	}

	// Filter out deleted domains
	var activeDomains []types.DomainDetails
	for _, domain := range m.ListDomainsOutput.Domains {
		if !m.deletedDomains[*domain.DomainId] {
			activeDomains = append(activeDomains, domain)
		}
	}
	return &sagemaker.ListDomainsOutput{Domains: activeDomains}, nil
}

func (m *mockedSageMakerStudio) DescribeDomain(ctx context.Context, params *sagemaker.DescribeDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeDomainOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DescribeDomainError != nil {
		return nil, m.DescribeDomainError
	}

	if m.deletedDomains[*params.DomainId] {
		return nil, &smithy.GenericAPIError{
			Code:    "ResourceNotFound",
			Message: "Domain not found",
		}
	}

	if status, exists := m.domainStatus[*params.DomainId]; exists {
		m.DescribeDomainOutput.Status = status
		return &m.DescribeDomainOutput, nil
	}

	return &m.DescribeDomainOutput, nil
}

func (m *mockedSageMakerStudio) DeleteDomain(ctx context.Context, params *sagemaker.DeleteDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteDomainOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedDomains[*params.DomainId] = true
	m.domainStatus[*params.DomainId] = types.DomainStatusDeleting
	return &m.DeleteDomainOutput, nil
}

func (m *mockedSageMakerStudio) ListUserProfiles(ctx context.Context, params *sagemaker.ListUserProfilesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListUserProfilesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Filter out deleted profiles
	var activeProfiles []types.UserProfileDetails
	for _, profile := range m.ListUserProfilesOutput.UserProfiles {
		key := *profile.DomainId + "/" + *profile.UserProfileName
		if !m.deletedProfiles[key] {
			activeProfiles = append(activeProfiles, profile)
		}
	}
	return &sagemaker.ListUserProfilesOutput{UserProfiles: activeProfiles}, nil
}

func (m *mockedSageMakerStudio) DescribeUserProfile(ctx context.Context, params *sagemaker.DescribeUserProfileInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeUserProfileOutput, error) {
	if m.DescribeUserProfileError != nil {
		return nil, m.DescribeUserProfileError
	}
	return &m.DescribeUserProfileOutput, nil
}

func (m *mockedSageMakerStudio) DeleteUserProfile(ctx context.Context, params *sagemaker.DeleteUserProfileInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteUserProfileOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteUserProfileError != nil {
		return nil, m.DeleteUserProfileError
	}

	key := *params.DomainId + "/" + *params.UserProfileName
	m.deletedProfiles[key] = true
	return &m.DeleteUserProfileOutput, nil
}

func (m *mockedSageMakerStudio) ListApps(ctx context.Context, params *sagemaker.ListAppsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListAppsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Filter out deleted apps and update status of deleted apps
	var activeApps []types.AppDetails
	for _, app := range m.ListAppsOutput.Apps {
		key := *app.DomainId + "/" + *app.UserProfileName + "/" + *app.AppName
		if m.deletedApps[key] {
			app.Status = types.AppStatusDeleted
		}
		activeApps = append(activeApps, app)
	}
	return &sagemaker.ListAppsOutput{Apps: activeApps}, nil
}

func (m *mockedSageMakerStudio) DescribeApp(ctx context.Context, params *sagemaker.DescribeAppInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeAppOutput, error) {
	if m.DescribeAppError != nil {
		return nil, m.DescribeAppError
	}
	return &m.DescribeAppOutput, nil
}

func (m *mockedSageMakerStudio) DeleteApp(ctx context.Context, params *sagemaker.DeleteAppInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteAppOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteAppError != nil {
		return nil, m.DeleteAppError
	}

	key := *params.DomainId + "/"
	if params.UserProfileName != nil {
		key += *params.UserProfileName
	} else if params.SpaceName != nil {
		key += *params.SpaceName
	}
	key += "/" + *params.AppName
	m.deletedApps[key] = true
	return &m.DeleteAppOutput, nil
}

func (m *mockedSageMakerStudio) ListMlflowTrackingServers(ctx context.Context, params *sagemaker.ListMlflowTrackingServersInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListMlflowTrackingServersOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ListMlflowTrackingServersError != nil {
		return nil, m.ListMlflowTrackingServersError
	}

	// Filter out deleted servers
	var activeServers []types.TrackingServerSummary
	for _, server := range m.ListMlflowTrackingServersOutput.TrackingServerSummaries {
		if !m.deletedServers[*server.TrackingServerName] {
			activeServers = append(activeServers, server)
		}
	}
	return &sagemaker.ListMlflowTrackingServersOutput{TrackingServerSummaries: activeServers}, nil
}

func (m *mockedSageMakerStudio) DeleteMlflowTrackingServer(ctx context.Context, params *sagemaker.DeleteMlflowTrackingServerInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteMlflowTrackingServerOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedServers[*params.TrackingServerName] = true
	return &m.DeleteMlflowTrackingServerOutput, nil
}

func (m *mockedSageMakerStudio) DescribeMlflowTrackingServer(ctx context.Context, params *sagemaker.DescribeMlflowTrackingServerInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeMlflowTrackingServerOutput, error) {
	return &m.DescribeMlflowTrackingServerOutput, nil
}

func (m *mockedSageMakerStudio) ListSpaces(ctx context.Context, params *sagemaker.ListSpacesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListSpacesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ListSpacesError != nil {
		return nil, m.ListSpacesError
	}

	// Filter out deleted spaces
	var activeSpaces []types.SpaceDetails
	for _, space := range m.ListSpacesOutput.Spaces {
		key := *space.DomainId + "/" + *space.SpaceName
		if !m.deletedSpaces[key] {
			activeSpaces = append(activeSpaces, space)
		}
	}
	return &sagemaker.ListSpacesOutput{Spaces: activeSpaces}, nil
}

func (m *mockedSageMakerStudio) DeleteSpace(ctx context.Context, params *sagemaker.DeleteSpaceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteSpaceOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteSpaceError != nil {
		return nil, m.DeleteSpaceError
	}

	key := *params.DomainId + "/" + *params.SpaceName
	m.deletedSpaces[key] = true
	return &m.DeleteSpaceOutput, nil
}

// testConfig holds test-specific timing configurations
type testConfig struct {
	retryInitialDelay time.Duration
	retryMaxDelay     time.Duration
	maxWaitTime       time.Duration
	maxRetries        int
}

// mockWaiter implements a fast mock waiter for testing
type mockWaiter struct {
	*mockedSageMakerStudio
	config testConfig
}

func (m *mockWaiter) waitForResourceDeletion(resourceName string, checkResource resourceChecker) error {
	delay := m.config.retryInitialDelay
	startTime := time.Now()

	for {
		if time.Since(startTime) > m.config.maxWaitTime {
			return fmt.Errorf("timeout waiting for %s to be deleted after %v", resourceName, m.config.maxWaitTime)
		}
		exists, err := checkResource()
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		time.Sleep(delay)
		if delay < m.config.retryMaxDelay {
			delay *= 2
		}
	}
}

// createTestSageMakerStudio creates a test instance with accelerated timeouts
func createTestSageMakerStudio(mockClient *mockedSageMakerStudio) *SageMakerStudio {
	waiter := &mockWaiter{
		mockedSageMakerStudio: mockClient,
		config: testConfig{
			retryInitialDelay: 100 * time.Millisecond,
			retryMaxDelay:     200 * time.Millisecond,
			maxWaitTime:       1 * time.Second,
			maxRetries:        2,
		},
	}

	return &SageMakerStudio{
		Client: waiter,
		Region: "us-east-1",
	}
}

func TestSageMakerStudio_GetAll(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name          string
		mockClient    *mockedSageMakerStudio
		config        config.Config
		expectedIds   []*string
		expectedError bool
	}{
		{
			name: "Basic successful case",
			mockClient: &mockedSageMakerStudio{
				ListDomainsOutput: sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{
						{
							DomainId:     aws.String("domain-1"),
							DomainName:   aws.String("test-domain-1"),
							CreationTime: &now,
						},
					},
				},
			},
			config: config.Config{
				SageMakerStudioDomain: config.SageMakerStudioDomainResourceType{},
			},
			expectedIds:   []*string{aws.String("domain-1")},
			expectedError: false,
		},
		{
			name: "Empty domains list",
			mockClient: &mockedSageMakerStudio{
				ListDomainsOutput: sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{},
				},
			},
			config: config.Config{
				SageMakerStudioDomain: config.SageMakerStudioDomainResourceType{},
			},
			expectedIds:   nil,
			expectedError: false,
		},
		{
			name: "Multiple domains",
			mockClient: &mockedSageMakerStudio{
				ListDomainsOutput: sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{
						{
							DomainId:     aws.String("domain-1"),
							DomainName:   aws.String("test-domain-1"),
							CreationTime: &now,
						},
						{
							DomainId:     aws.String("domain-2"),
							DomainName:   aws.String("test-domain-2"),
							CreationTime: &now,
						},
					},
				},
			},
			config: config.Config{
				SageMakerStudioDomain: config.SageMakerStudioDomainResourceType{},
			},
			expectedIds:   []*string{aws.String("domain-1"), aws.String("domain-2")},
			expectedError: false,
		},
		{
			name: "Error listing domains",
			mockClient: &mockedSageMakerStudio{
				ListDomainsOutput: sagemaker.ListDomainsOutput{},
				ListDomainsError:  fmt.Errorf("AWS API error"),
			},
			config: config.Config{
				SageMakerStudioDomain: config.SageMakerStudioDomainResourceType{},
			},
			expectedIds:   nil,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			studio := createTestSageMakerStudio(tc.mockClient)
			domains, err := studio.getAll(context.Background(), tc.config)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedIds, domains)
			}
		})
	}
}

func TestSageMakerStudio_NukeAll(t *testing.T) {
	testCases := []struct {
		name          string
		mockClient    *mockedSageMakerStudio
		domains       []*string
		expectedError bool
	}{
		{
			name: "Successfully delete domain with apps and user profiles",
			mockClient: func() *mockedSageMakerStudio {
				mock := newMockedSageMakerStudio()
				mock.ListAppsOutput = sagemaker.ListAppsOutput{
					Apps: []types.AppDetails{
						{
							AppName:         aws.String("app-1"),
							AppType:         types.AppTypeJupyterServer,
							DomainId:        aws.String("domain-1"),
							UserProfileName: aws.String("user-1"),
							Status:          types.AppStatusInService,
						},
					},
				}
				mock.ListUserProfilesOutput = sagemaker.ListUserProfilesOutput{
					UserProfiles: []types.UserProfileDetails{
						{
							DomainId:        aws.String("domain-1"),
							UserProfileName: aws.String("user-1"),
						},
					},
				}
				return mock
			}(),
			domains:       []*string{aws.String("domain-1")},
			expectedError: false,
		},
		{
			name:          "Empty domains list",
			mockClient:    newMockedSageMakerStudio(),
			domains:       []*string{},
			expectedError: false,
		},
		{
			name: "Error deleting app",
			mockClient: func() *mockedSageMakerStudio {
				mock := newMockedSageMakerStudio()
				mock.ListAppsOutput = sagemaker.ListAppsOutput{
					Apps: []types.AppDetails{
						{
							AppName:         aws.String("app-1"),
							AppType:         types.AppTypeJupyterServer,
							DomainId:        aws.String("domain-1"),
							UserProfileName: aws.String("user-1"),
							Status:          types.AppStatusInService,
						},
					},
				}
				mock.DeleteAppError = fmt.Errorf("Failed to delete app")
				mock.ListUserProfilesOutput = sagemaker.ListUserProfilesOutput{
					UserProfiles: []types.UserProfileDetails{
						{
							DomainId:        aws.String("domain-1"),
							UserProfileName: aws.String("user-1"),
						},
					},
				}
				mock.DescribeAppOutput = sagemaker.DescribeAppOutput{
					AppArn:          aws.String("arn:aws:sagemaker:us-east-1:123456789012:app/domain-1/user-1/app-1"),
					AppName:         aws.String("app-1"),
					AppType:         types.AppTypeJupyterServer,
					DomainId:        aws.String("domain-1"),
					UserProfileName: aws.String("user-1"),
					Status:          types.AppStatusInService,
				}
				mock.domainStatus = map[string]types.DomainStatus{
					"domain-1": types.DomainStatusInService,
				}
				mock.DescribeDomainOutput = sagemaker.DescribeDomainOutput{
					DomainId: aws.String("domain-1"),
					Status:   types.DomainStatusInService,
				}
				return mock
			}(),
			domains:       []*string{aws.String("domain-1")},
			expectedError: true,
		},
		{
			name: "Error deleting user profile",
			mockClient: func() *mockedSageMakerStudio {
				mock := newMockedSageMakerStudio()
				mock.ListUserProfilesOutput = sagemaker.ListUserProfilesOutput{
					UserProfiles: []types.UserProfileDetails{
						{
							DomainId:        aws.String("domain-1"),
							UserProfileName: aws.String("user-1"),
						},
					},
				}
				mock.DeleteUserProfileError = fmt.Errorf("Failed to delete user profile")
				return mock
			}(),
			domains:       []*string{aws.String("domain-1")},
			expectedError: true,
		},
		{
			name: "Domain with MLflow tracking server",
			mockClient: func() *mockedSageMakerStudio {
				mock := newMockedSageMakerStudio()
				mock.ListMlflowTrackingServersOutput = sagemaker.ListMlflowTrackingServersOutput{
					TrackingServerSummaries: []types.TrackingServerSummary{
						{
							TrackingServerName:   aws.String("server-1"),
							TrackingServerStatus: types.TrackingServerStatusCreated,
						},
					},
				}
				mock.ListDomainsOutput = sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{
						{
							DomainId: aws.String("domain-1"),
						},
					},
				}
				return mock
			}(),
			domains:       []*string{aws.String("domain-1")},
			expectedError: false,
		},
		{
			name: "Domain with spaces",
			mockClient: func() *mockedSageMakerStudio {
				mock := newMockedSageMakerStudio()
				mock.ListSpacesOutput = sagemaker.ListSpacesOutput{
					Spaces: []types.SpaceDetails{
						{
							SpaceName: aws.String("space-1"),
							DomainId:  aws.String("domain-1"),
							Status:    types.SpaceStatusInService,
						},
					},
				}
				mock.ListDomainsOutput = sagemaker.ListDomainsOutput{
					Domains: []types.DomainDetails{
						{
							DomainId: aws.String("domain-1"),
							Status:   types.DomainStatusInService,
						},
					},
				}
				return mock
			}(),
			domains:       []*string{aws.String("domain-1")},
			expectedError: false,
		},
		{
			name: "Error deleting space",
			mockClient: func() *mockedSageMakerStudio {
				mock := newMockedSageMakerStudio()
				mock.ListSpacesOutput = sagemaker.ListSpacesOutput{
					Spaces: []types.SpaceDetails{
						{
							SpaceName: aws.String("space-1"),
							DomainId:  aws.String("domain-1"),
							Status:    types.SpaceStatusInService,
						},
					},
				}
				mock.DeleteSpaceError = fmt.Errorf("Failed to delete space")
				return mock
			}(),
			domains:       []*string{aws.String("domain-1")},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Enable parallel test execution
			studio := createTestSageMakerStudio(tc.mockClient)
			err := studio.nukeAll(aws.ToStringSlice(tc.domains))

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
