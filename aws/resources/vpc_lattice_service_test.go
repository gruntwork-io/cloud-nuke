package resources_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedVPCLatticeService struct {
	resources.VPCLatticeServiceAPI

	ListServicesOutput  vpclattice.ListServicesOutput
	DeleteServiceOutput vpclattice.DeleteServiceOutput

	ListServiceNetworkServiceAssociationsOutput  vpclattice.ListServiceNetworkServiceAssociationsOutput
	DeleteServiceNetworkServiceAssociationOutput vpclattice.DeleteServiceNetworkServiceAssociationOutput
}

func (m mockedVPCLatticeService) ListServices(ctx context.Context, params *vpclattice.ListServicesInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m mockedVPCLatticeService) DeleteService(ctx context.Context, params *vpclattice.DeleteServiceInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}
func (m mockedVPCLatticeService) ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error) {
	return &m.ListServiceNetworkServiceAssociationsOutput, nil
}
func (m mockedVPCLatticeService) DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error) {
	return &m.DeleteServiceNetworkServiceAssociationOutput, nil
}

func TestVPCLatticeService_GetAll(t *testing.T) {

	t.Parallel()

	var (
		id1 = "aws-nuke-test-" + util.UniqueID()
		id2 = "aws-nuke-test-" + util.UniqueID()
		now = time.Now()
	)

	obj := resources.VPCLatticeService{
		Client: mockedVPCLatticeService{
			ListServicesOutput: vpclattice.ListServicesOutput{
				Items: []types.ServiceSummary{
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
				VPCLatticeService: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, names)
		})
	}
}

func TestVPCLatticeService_NukeAll(t *testing.T) {
	t.Parallel()

	obj := resources.VPCLatticeService{
		Client: mockedVPCLatticeService{
			ListServicesOutput: vpclattice.ListServicesOutput{},
		},
	}
	err := obj.Nuke([]string{"test"})
	require.NoError(t, err)
}
