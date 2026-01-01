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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVPCLatticeTargetGroupClient struct {
	ListTargetGroupsOutput  vpclattice.ListTargetGroupsOutput
	ListTargetsOutput       vpclattice.ListTargetsOutput
	DeregisterTargetsOutput vpclattice.DeregisterTargetsOutput
	DeleteTargetGroupOutput vpclattice.DeleteTargetGroupOutput
}

func (m *mockVPCLatticeTargetGroupClient) ListTargetGroups(ctx context.Context, params *vpclattice.ListTargetGroupsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTargetGroupsOutput, error) {
	return &m.ListTargetGroupsOutput, nil
}

func (m *mockVPCLatticeTargetGroupClient) ListTargets(ctx context.Context, params *vpclattice.ListTargetsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTargetsOutput, error) {
	return &m.ListTargetsOutput, nil
}

func (m *mockVPCLatticeTargetGroupClient) DeregisterTargets(ctx context.Context, params *vpclattice.DeregisterTargetsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeregisterTargetsOutput, error) {
	return &m.DeregisterTargetsOutput, nil
}

func (m *mockVPCLatticeTargetGroupClient) DeleteTargetGroup(ctx context.Context, params *vpclattice.DeleteTargetGroupInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteTargetGroupOutput, error) {
	return &m.DeleteTargetGroupOutput, nil
}

func TestVPCLatticeTargetGroup_ResourceName(t *testing.T) {
	r := NewVPCLatticeTargetGroup()
	assert.Equal(t, "vpc-lattice-target-group", r.ResourceName())
}

func TestVPCLatticeTargetGroup_MaxBatchSize(t *testing.T) {
	r := NewVPCLatticeTargetGroup()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListVPCLatticeTargetGroups(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockVPCLatticeTargetGroupClient{
		ListTargetGroupsOutput: vpclattice.ListTargetGroupsOutput{
			Items: []types.TargetGroupSummary{
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"), Name: aws.String("tg-1"), CreatedAt: aws.Time(now)},
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-2"), Name: aws.String("tg-2"), CreatedAt: aws.Time(now)},
			},
		},
	}

	ids, err := listVPCLatticeTargetGroups(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1",
		"arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-2",
	}, aws.ToStringSlice(ids))
}

func TestListVPCLatticeTargetGroups_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockVPCLatticeTargetGroupClient{
		ListTargetGroupsOutput: vpclattice.ListTargetGroupsOutput{
			Items: []types.TargetGroupSummary{
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"), Name: aws.String("tg-1"), CreatedAt: aws.Time(now)},
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/skip-tg"), Name: aws.String("skip-tg"), CreatedAt: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	ids, err := listVPCLatticeTargetGroups(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"}, aws.ToStringSlice(ids))
}

func TestDeleteVPCLatticeTargetGroup(t *testing.T) {
	t.Parallel()

	mock := &mockVPCLatticeTargetGroupClient{
		ListTargetsOutput: vpclattice.ListTargetsOutput{
			Items: []types.TargetSummary{
				{Id: aws.String("target-1")},
			},
		},
	}

	err := deleteVPCLatticeTargetGroup(context.Background(), mock, aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"))
	require.NoError(t, err)
}
