package resources_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/vpclattice"
	"github.com/aws/aws-sdk-go/service/vpclattice/vpclatticeiface"
	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedVPCLatticeServiceNetwork struct {
	vpclatticeiface.VPCLatticeAPI
	DeleteServiceNetworkOutput vpclattice.DeleteServiceNetworkOutput
	ListServiceNetworksOutput  vpclattice.ListServiceNetworksOutput
}

func (m mockedVPCLatticeServiceNetwork) ListServiceNetworksWithContext(aws.Context, *vpclattice.ListServiceNetworksInput, ...request.Option) (*vpclattice.ListServiceNetworksOutput, error) {
	return &m.ListServiceNetworksOutput, nil
}

func (m mockedVPCLatticeServiceNetwork) DeleteServiceNetworkWithContext(aws.Context, *vpclattice.DeleteServiceNetworkInput, ...request.Option) (*vpclattice.DeleteServiceNetworkOutput, error) {
	return &m.DeleteServiceNetworkOutput, nil
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
				Items: []*vpclattice.ServiceNetworkSummary{
					{
						Arn:       awsgo.String(id1),
						Name:      awsgo.String(id1),
						CreatedAt: aws.Time(now),
					}, {
						Arn:       awsgo.String(id2),
						Name:      awsgo.String(id2),
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
					TimeAfter: awsgo.Time(now),
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
	err := obj.Nuke([]string{"test"})
	require.NoError(t, err)
}
