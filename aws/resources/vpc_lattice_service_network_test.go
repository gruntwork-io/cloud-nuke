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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVPCLatticeServiceNetworkClient struct {
	ListServiceNetworksOutput                    vpclattice.ListServiceNetworksOutput
	DeleteServiceNetworkOutput                   vpclattice.DeleteServiceNetworkOutput
	ListServiceNetworkServiceAssociationsOutput  vpclattice.ListServiceNetworkServiceAssociationsOutput
	DeleteServiceNetworkServiceAssociationOutput vpclattice.DeleteServiceNetworkServiceAssociationOutput
}

func (m *mockVPCLatticeServiceNetworkClient) ListServiceNetworks(ctx context.Context, params *vpclattice.ListServiceNetworksInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworksOutput, error) {
	return &m.ListServiceNetworksOutput, nil
}

func (m *mockVPCLatticeServiceNetworkClient) DeleteServiceNetwork(ctx context.Context, params *vpclattice.DeleteServiceNetworkInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkOutput, error) {
	return &m.DeleteServiceNetworkOutput, nil
}

func (m *mockVPCLatticeServiceNetworkClient) ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error) {
	return &m.ListServiceNetworkServiceAssociationsOutput, nil
}

func (m *mockVPCLatticeServiceNetworkClient) DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error) {
	return &m.DeleteServiceNetworkServiceAssociationOutput, nil
}

func TestVPCLatticeServiceNetwork_ResourceName(t *testing.T) {
	r := NewVPCLatticeServiceNetwork()
	assert.Equal(t, "vpc-lattice-service-network", r.ResourceName())
}

func TestVPCLatticeServiceNetwork_MaxBatchSize(t *testing.T) {
	r := NewVPCLatticeServiceNetwork()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListVPCLatticeServiceNetworks(t *testing.T) {
	t.Parallel()

	id1 := "aws-nuke-test-" + util.UniqueID()
	id2 := "aws-nuke-test-" + util.UniqueID()
	now := time.Now()

	mock := &mockVPCLatticeServiceNetworkClient{
		ListServiceNetworksOutput: vpclattice.ListServiceNetworksOutput{
			Items: []types.ServiceNetworkSummary{
				{
					Arn:       aws.String(id1),
					Name:      aws.String(id1),
					CreatedAt: aws.Time(now),
				},
				{
					Arn:       aws.String(id2),
					Name:      aws.String(id2),
					CreatedAt: aws.Time(now.Add(1 * time.Hour)),
				},
			},
		},
	}

	ids, err := listVPCLatticeServiceNetworks(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{id1, id2}, aws.ToStringSlice(ids))
}

func TestListVPCLatticeServiceNetworks_WithNameExclusionFilter(t *testing.T) {
	t.Parallel()

	id1 := "aws-nuke-test-" + util.UniqueID()
	id2 := "aws-nuke-test-" + util.UniqueID()
	now := time.Now()

	mock := &mockVPCLatticeServiceNetworkClient{
		ListServiceNetworksOutput: vpclattice.ListServiceNetworksOutput{
			Items: []types.ServiceNetworkSummary{
				{
					Arn:       aws.String(id1),
					Name:      aws.String(id1),
					CreatedAt: aws.Time(now),
				},
				{
					Arn:       aws.String(id2),
					Name:      aws.String(id2),
					CreatedAt: aws.Time(now.Add(1 * time.Hour)),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(id2)}},
		},
	}

	ids, err := listVPCLatticeServiceNetworks(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{id1}, aws.ToStringSlice(ids))
}

func TestListVPCLatticeServiceNetworks_WithTimeAfterExclusionFilter(t *testing.T) {
	t.Parallel()

	id1 := "aws-nuke-test-" + util.UniqueID()
	id2 := "aws-nuke-test-" + util.UniqueID()
	now := time.Now()

	mock := &mockVPCLatticeServiceNetworkClient{
		ListServiceNetworksOutput: vpclattice.ListServiceNetworksOutput{
			Items: []types.ServiceNetworkSummary{
				{
					Arn:       aws.String(id1),
					Name:      aws.String(id1),
					CreatedAt: aws.Time(now),
				},
				{
					Arn:       aws.String(id2),
					Name:      aws.String(id2),
					CreatedAt: aws.Time(now.Add(1 * time.Hour)),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now),
		},
	}

	ids, err := listVPCLatticeServiceNetworks(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{id1}, aws.ToStringSlice(ids))
}

func TestDeleteVPCLatticeServiceNetwork(t *testing.T) {
	t.Parallel()

	mock := &mockVPCLatticeServiceNetworkClient{
		ListServiceNetworkServiceAssociationsOutput: vpclattice.ListServiceNetworkServiceAssociationsOutput{
			Items: []types.ServiceNetworkServiceAssociationSummary{},
		},
	}

	err := deleteVPCLatticeServiceNetwork(context.Background(), mock, aws.String("test-arn"))
	require.NoError(t, err)
}
