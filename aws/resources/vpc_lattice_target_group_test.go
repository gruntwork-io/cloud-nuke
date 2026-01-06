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

func (m *mockVPCLatticeTargetGroupClient) ListTargetGroups(_ context.Context, _ *vpclattice.ListTargetGroupsInput, _ ...func(*vpclattice.Options)) (*vpclattice.ListTargetGroupsOutput, error) {
	return &m.ListTargetGroupsOutput, nil
}

func (m *mockVPCLatticeTargetGroupClient) ListTargets(_ context.Context, _ *vpclattice.ListTargetsInput, _ ...func(*vpclattice.Options)) (*vpclattice.ListTargetsOutput, error) {
	return &m.ListTargetsOutput, nil
}

func (m *mockVPCLatticeTargetGroupClient) DeregisterTargets(_ context.Context, _ *vpclattice.DeregisterTargetsInput, _ ...func(*vpclattice.Options)) (*vpclattice.DeregisterTargetsOutput, error) {
	return &m.DeregisterTargetsOutput, nil
}

func (m *mockVPCLatticeTargetGroupClient) DeleteTargetGroup(_ context.Context, _ *vpclattice.DeleteTargetGroupInput, _ ...func(*vpclattice.Options)) (*vpclattice.DeleteTargetGroupOutput, error) {
	return &m.DeleteTargetGroupOutput, nil
}

func TestVPCLatticeTargetGroup_Properties(t *testing.T) {
	t.Parallel()

	r := NewVPCLatticeTargetGroup()
	assert.Equal(t, "vpc-lattice-target-group", r.ResourceName())
	assert.Equal(t, DefaultBatchSize, r.MaxBatchSize())
}

func TestListVPCLatticeTargetGroups(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name     string
		items    []types.TargetGroupSummary
		cfg      config.ResourceType
		expected []string
	}{
		{
			name: "returns all target groups without filter",
			items: []types.TargetGroupSummary{
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"), Name: aws.String("tg-1"), CreatedAt: aws.Time(now)},
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-2"), Name: aws.String("tg-2"), CreatedAt: aws.Time(now)},
			},
			cfg:      config.ResourceType{},
			expected: []string{"arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1", "arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-2"},
		},
		{
			name: "filters target groups by exclude rule",
			items: []types.TargetGroupSummary{
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"), Name: aws.String("tg-1"), CreatedAt: aws.Time(now)},
				{Arn: aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/skip-tg"), Name: aws.String("skip-tg"), CreatedAt: aws.Time(now)},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
				},
			},
			expected: []string{"arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"},
		},
		{
			name:     "handles empty result",
			items:    []types.TargetGroupSummary{},
			cfg:      config.ResourceType{},
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := &mockVPCLatticeTargetGroupClient{
				ListTargetGroupsOutput: vpclattice.ListTargetGroupsOutput{Items: tc.items},
			}

			ids, err := listVPCLatticeTargetGroups(context.Background(), mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeregisterVPCLatticeTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets []types.TargetSummary
	}{
		{
			name:    "deregisters targets when present",
			targets: []types.TargetSummary{{Id: aws.String("target-1")}, {Id: aws.String("target-2")}},
		},
		{
			name:    "handles no targets",
			targets: []types.TargetSummary{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := &mockVPCLatticeTargetGroupClient{
				ListTargetsOutput: vpclattice.ListTargetsOutput{Items: tc.targets},
			}

			err := deregisterVPCLatticeTargets(context.Background(), mock, aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"))
			require.NoError(t, err)
		})
	}
}

func TestDeleteVPCLatticeTargetGroup(t *testing.T) {
	t.Parallel()

	mock := &mockVPCLatticeTargetGroupClient{}
	err := deleteVPCLatticeTargetGroup(context.Background(), mock, aws.String("arn:aws:vpc-lattice:us-east-1:123456789012:targetgroup/tg-1"))
	require.NoError(t, err)
}
