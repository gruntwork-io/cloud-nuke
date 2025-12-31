package resources_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedVPCLatticeServiceNetwork struct {
	resources.VPCLatticeServiceNetworkAPI
	DeleteServiceNetworkOutput vpclattice.DeleteServiceNetworkOutput
	ListServiceNetworksOutput  vpclattice.ListServiceNetworksOutput

	ListServiceNetworkServiceAssociationsOutput  vpclattice.ListServiceNetworkServiceAssociationsOutput
	DeleteServiceNetworkServiceAssociationOutput vpclattice.DeleteServiceNetworkServiceAssociationOutput
}

func (m mockedVPCLatticeServiceNetwork) ListServiceNetworks(ctx context.Context, params *vpclattice.ListServiceNetworksInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworksOutput, error) {
	return &m.ListServiceNetworksOutput, nil
}

func (m mockedVPCLatticeServiceNetwork) DeleteServiceNetwork(ctx context.Context, params *vpclattice.DeleteServiceNetworkInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkOutput, error) {
	return &m.DeleteServiceNetworkOutput, nil
}

func (m mockedVPCLatticeServiceNetwork) ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error) {
	return &m.ListServiceNetworkServiceAssociationsOutput, nil
}
func (m mockedVPCLatticeServiceNetwork) DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error) {
	return &m.DeleteServiceNetworkServiceAssociationOutput, nil
}

func TestVPCLatticeServiceNetwork_GetAll(t *testing.T) {

	t.Parallel()

	var (
		id1 = "aws-nuke-test-" + util.UniqueID()
		id2 = "aws-nuke-test-" + util.UniqueID()
		now = time.Now()
	)

	obj := resources.VPCLatticeServiceNetwork{
		Client: mockedVPCLatticeServiceNetwork{
			ListServiceNetworksOutput: vpclattice.ListServiceNetworksOutput{
				Items: []types.ServiceNetworkSummary{
					{
						Arn:       aws.String(id1),
						Name:      aws.String(id1),
						CreatedAt: aws.Time(now),
					}, {
						Arn:       aws.String(id2),
						Name:      aws.String(id2),
						CreatedAt: aws.Time(now.Add(1 * time.Hour)),
					},
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
			expected:  []string{id1, id2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(id2),
					}}},
			},
			expected: []string{id1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{id1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := obj.GetAndSetIdentifiers(context.Background(), config.Config{
				VPCLatticeServiceNetwork: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, names)
		})
	}
}

func TestVPCLatticeServiceNetwork__NukeAll(t *testing.T) {
	t.Parallel()

	obj := resources.VPCLatticeServiceNetwork{
		Client: mockedVPCLatticeServiceNetwork{
			ListServiceNetworksOutput: vpclattice.ListServiceNetworksOutput{},
		},
	}
	err := obj.Nuke(context.TODO(), []string{"test"})
	require.NoError(t, err)
}
