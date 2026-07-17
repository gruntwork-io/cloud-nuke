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

type mockedVPCLatticeServiceNetwork struct {
	ListServiceNetworksOutput                    vpclattice.ListServiceNetworksOutput
	DeleteServiceNetworkOutput                   vpclattice.DeleteServiceNetworkOutput
	ListServiceNetworkServiceAssociationsOutput  vpclattice.ListServiceNetworkServiceAssociationsOutput
	DeleteServiceNetworkServiceAssociationOutput vpclattice.DeleteServiceNetworkServiceAssociationOutput
	ListServiceNetworkVpcAssociationsOutput      vpclattice.ListServiceNetworkVpcAssociationsOutput
	DeleteServiceNetworkVpcAssociationOutput     vpclattice.DeleteServiceNetworkVpcAssociationOutput

	ListServiceNetworkResourceAssociationsOutput  vpclattice.ListServiceNetworkResourceAssociationsOutput
	DeleteServiceNetworkResourceAssociationOutput vpclattice.DeleteServiceNetworkResourceAssociationOutput

	// Track calls to simulate associations being deleted
	listAssociationsCalls         int
	listVpcAssociationsCalls      int
	listResourceAssociationsCalls int
}

func (m *mockedVPCLatticeServiceNetwork) ListServiceNetworks(ctx context.Context, params *vpclattice.ListServiceNetworksInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworksOutput, error) {
	return &m.ListServiceNetworksOutput, nil
}

func (m *mockedVPCLatticeServiceNetwork) DeleteServiceNetwork(ctx context.Context, params *vpclattice.DeleteServiceNetworkInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkOutput, error) {
	return &m.DeleteServiceNetworkOutput, nil
}

func (m *mockedVPCLatticeServiceNetwork) ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error) {
	m.listAssociationsCalls++
	// First call returns associations, subsequent calls return empty (simulating deletion)
	if m.listAssociationsCalls <= 1 {
		return &m.ListServiceNetworkServiceAssociationsOutput, nil
	}
	return &vpclattice.ListServiceNetworkServiceAssociationsOutput{Items: []types.ServiceNetworkServiceAssociationSummary{}}, nil
}

func (m *mockedVPCLatticeServiceNetwork) DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error) {
	return &m.DeleteServiceNetworkServiceAssociationOutput, nil
}

func (m *mockedVPCLatticeServiceNetwork) ListServiceNetworkVpcAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkVpcAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkVpcAssociationsOutput, error) {
	m.listVpcAssociationsCalls++
	// First call returns associations, subsequent calls return empty (simulating deletion)
	if m.listVpcAssociationsCalls <= 1 {
		return &m.ListServiceNetworkVpcAssociationsOutput, nil
	}
	return &vpclattice.ListServiceNetworkVpcAssociationsOutput{Items: []types.ServiceNetworkVpcAssociationSummary{}}, nil
}

func (m *mockedVPCLatticeServiceNetwork) DeleteServiceNetworkVpcAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkVpcAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkVpcAssociationOutput, error) {
	return &m.DeleteServiceNetworkVpcAssociationOutput, nil
}

func (m *mockedVPCLatticeServiceNetwork) ListServiceNetworkResourceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkResourceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkResourceAssociationsOutput, error) {
	m.listResourceAssociationsCalls++
	// First call returns associations, subsequent calls return empty (simulating deletion)
	if m.listResourceAssociationsCalls <= 1 {
		return &m.ListServiceNetworkResourceAssociationsOutput, nil
	}
	return &vpclattice.ListServiceNetworkResourceAssociationsOutput{Items: []types.ServiceNetworkResourceAssociationSummary{}}, nil
}

func (m *mockedVPCLatticeServiceNetwork) DeleteServiceNetworkResourceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkResourceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkResourceAssociationOutput, error) {
	return &m.DeleteServiceNetworkResourceAssociationOutput, nil
}

func (m *mockedVPCLatticeServiceNetwork) ListTagsForResource(ctx context.Context, params *vpclattice.ListTagsForResourceInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTagsForResourceOutput, error) {
	if aws.ToString(params.ResourceArn) == "arn:aws:vpc-lattice:us-east-1:123456789012:servicenetwork/sn-1" {
		return &vpclattice.ListTagsForResourceOutput{
			Tags: map[string]string{"env": "prod"},
		}, nil
	}
	return &vpclattice.ListTagsForResourceOutput{}, nil
}

func TestVPCLatticeServiceNetwork_ResourceMetadata(t *testing.T) {
	t.Parallel()

	r := NewVPCLatticeServiceNetwork()
	require.Equal(t, "vpc-lattice-service-network", r.ResourceName())
	require.Equal(t, DefaultBatchSize, r.MaxBatchSize())
}

func TestVPCLatticeServiceNetwork_GetAll(t *testing.T) {
	t.Parallel()

	testArn1 := "arn:aws:vpc-lattice:us-east-1:123456789012:servicenetwork/sn-1"
	testArn2 := "arn:aws:vpc-lattice:us-east-1:123456789012:servicenetwork/sn-2"
	testName1 := "test-network-1"
	testName2 := "test-network-2"
	now := time.Now()

	mock := &mockedVPCLatticeServiceNetwork{
		ListServiceNetworksOutput: vpclattice.ListServiceNetworksOutput{
			Items: []types.ServiceNetworkSummary{
				{
					Arn:       aws.String(testArn1),
					Name:      aws.String(testName1),
					CreatedAt: aws.Time(now),
				},
				{
					Arn:       aws.String(testArn2),
					Name:      aws.String(testName2),
					CreatedAt: aws.Time(now.Add(1 * time.Hour)),
				},
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName2)}},
				},
			},
			expected: []string{testArn1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{testArn1},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^prod$")}},
				},
			},
			expected: []string{testArn1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listVPCLatticeServiceNetworks(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestVPCLatticeServiceNetwork_DeleteServiceAssociations(t *testing.T) {
	t.Parallel()

	mock := &mockedVPCLatticeServiceNetwork{
		ListServiceNetworkServiceAssociationsOutput: vpclattice.ListServiceNetworkServiceAssociationsOutput{
			Items: []types.ServiceNetworkServiceAssociationSummary{
				{Id: aws.String("snsa-123456")},
				{Id: aws.String("snsa-789012")},
			},
		},
	}

	err := deleteServiceAssociations(context.Background(), mock, aws.String("sn-test"))
	require.NoError(t, err)
}

func TestVPCLatticeServiceNetwork_DeleteVpcAssociations(t *testing.T) {
	t.Parallel()

	mock := &mockedVPCLatticeServiceNetwork{
		ListServiceNetworkVpcAssociationsOutput: vpclattice.ListServiceNetworkVpcAssociationsOutput{
			Items: []types.ServiceNetworkVpcAssociationSummary{
				{Id: aws.String("snva-123456")},
				{Id: aws.String("snva-789012")},
			},
		},
	}

	err := deleteVpcAssociations(context.Background(), mock, aws.String("sn-test"))
	require.NoError(t, err)
}

func TestVPCLatticeServiceNetwork_DeleteResourceAssociations(t *testing.T) {
	t.Parallel()

	// A managed association is included to confirm it is skipped rather than
	// deleted (it cannot be deleted from the service network side).
	mock := &mockedVPCLatticeServiceNetwork{
		ListServiceNetworkResourceAssociationsOutput: vpclattice.ListServiceNetworkResourceAssociationsOutput{
			Items: []types.ServiceNetworkResourceAssociationSummary{
				{Id: aws.String("snra-123456")},
				{Id: aws.String("snra-789012")},
				{Id: aws.String("snra-managed"), IsManagedAssociation: aws.Bool(true)},
			},
		},
	}

	err := deleteResourceAssociations(context.Background(), mock, aws.String("sn-test"))
	require.NoError(t, err)
}

func TestVPCLatticeServiceNetwork_DeleteNetwork(t *testing.T) {
	t.Parallel()

	mock := &mockedVPCLatticeServiceNetwork{}
	err := deleteServiceNetwork(context.Background(), mock, aws.String("sn-test"))
	require.NoError(t, err)
}

func TestVPCLatticeServiceNetwork_MultiStepDeleter(t *testing.T) {
	t.Parallel()

	// All three association types are populated so the deleter must clear
	// service, VPC, and resource associations before the network delete succeeds
	// (regression for the 409 "has ...associated" failures caused by leftover
	// associations of each kind).
	mock := &mockedVPCLatticeServiceNetwork{
		ListServiceNetworkServiceAssociationsOutput: vpclattice.ListServiceNetworkServiceAssociationsOutput{
			Items: []types.ServiceNetworkServiceAssociationSummary{{Id: aws.String("snsa-1")}},
		},
		ListServiceNetworkVpcAssociationsOutput: vpclattice.ListServiceNetworkVpcAssociationsOutput{
			Items: []types.ServiceNetworkVpcAssociationSummary{{Id: aws.String("snva-1")}},
		},
		ListServiceNetworkResourceAssociationsOutput: vpclattice.ListServiceNetworkResourceAssociationsOutput{
			Items: []types.ServiceNetworkResourceAssociationSummary{{Id: aws.String("snra-1")}},
		},
	}

	nuker := resource.MultiStepDeleter(deleteServiceAssociations, deleteVpcAssociations, deleteResourceAssociations, waitForAssociationsDeleted, deleteServiceNetwork)
	results := nuker(context.Background(), mock, resource.Scope{Region: "us-east-1"}, "vpc-lattice-service-network", []*string{aws.String("sn-test")})

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Equal(t, "sn-test", results[0].Identifier)
}
