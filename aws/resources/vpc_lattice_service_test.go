package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockVPCLatticeServiceClient struct {
	ListServicesOutput                           vpclattice.ListServicesOutput
	DeleteServiceOutput                          vpclattice.DeleteServiceOutput
	ListServiceNetworkServiceAssociationsOutput  vpclattice.ListServiceNetworkServiceAssociationsOutput
	DeleteServiceNetworkServiceAssociationOutput vpclattice.DeleteServiceNetworkServiceAssociationOutput
}

func (m *mockVPCLatticeServiceClient) ListServices(ctx context.Context, params *vpclattice.ListServicesInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m *mockVPCLatticeServiceClient) DeleteService(ctx context.Context, params *vpclattice.DeleteServiceInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m *mockVPCLatticeServiceClient) ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error) {
	return &m.ListServiceNetworkServiceAssociationsOutput, nil
}

func (m *mockVPCLatticeServiceClient) DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error) {
	return &m.DeleteServiceNetworkServiceAssociationOutput, nil
}

func TestListVPCLatticeServices(t *testing.T) {
	t.Parallel()

	testName1 := "test-service-1"
	testName2 := "test-service-2"
	now := time.Now()

	mock := &mockVPCLatticeServiceClient{
		ListServicesOutput: vpclattice.ListServicesOutput{
			Items: []types.ServiceSummary{
				{Arn: aws.String(testName1), Name: aws.String(testName1), CreatedAt: aws.Time(now)},
				{Arn: aws.String(testName2), Name: aws.String(testName2), CreatedAt: aws.Time(now.Add(1 * time.Hour))},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName2)}},
				},
			},
			expected: []string{testName1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listVPCLatticeServices(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteVPCLatticeServiceAssociations(t *testing.T) {
	t.Parallel()

	mock := &mockVPCLatticeServiceClient{
		ListServiceNetworkServiceAssociationsOutput: vpclattice.ListServiceNetworkServiceAssociationsOutput{
			Items: []types.ServiceNetworkServiceAssociationSummary{
				{Id: aws.String("assoc-1")},
				{Id: aws.String("assoc-2")},
			},
		},
	}

	err := deleteVPCLatticeServiceAssociations(context.Background(), mock, aws.String("test-service"))
	require.NoError(t, err)
}

func TestDeleteVPCLatticeService(t *testing.T) {
	t.Parallel()

	mock := &mockVPCLatticeServiceClient{}

	err := deleteVPCLatticeService(context.Background(), mock, aws.String("test-service"))
	require.NoError(t, err)
}

func TestWaitForVPCLatticeServiceAssociationsDeleted(t *testing.T) {
	t.Parallel()

	// Test with no associations (should return immediately)
	mock := &mockVPCLatticeServiceClient{
		ListServiceNetworkServiceAssociationsOutput: vpclattice.ListServiceNetworkServiceAssociationsOutput{
			Items: []types.ServiceNetworkServiceAssociationSummary{},
		},
	}

	err := waitForVPCLatticeServiceAssociationsDeleted(context.Background(), mock, aws.String("test-service"))
	require.NoError(t, err)
}
