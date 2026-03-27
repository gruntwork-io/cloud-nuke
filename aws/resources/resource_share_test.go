package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/ram/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockRAMResourceShareClient struct {
	getOutputs  []ram.GetResourceSharesOutput
	getErr      error
	getCall     int
	deleteErr   error
	deletedARNs []string
}

func (m *mockRAMResourceShareClient) GetResourceShares(_ context.Context, _ *ram.GetResourceSharesInput, _ ...func(*ram.Options)) (*ram.GetResourceSharesOutput, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	if len(m.getOutputs) == 0 {
		return &ram.GetResourceSharesOutput{}, nil
	}

	if m.getCall >= len(m.getOutputs) {
		last := m.getOutputs[len(m.getOutputs)-1]
		return &last, nil
	}

	output := m.getOutputs[m.getCall]
	m.getCall++
	return &output, nil
}

func (m *mockRAMResourceShareClient) DeleteResourceShare(_ context.Context, params *ram.DeleteResourceShareInput, _ ...func(*ram.Options)) (*ram.DeleteResourceShareOutput, error) {
	m.deletedARNs = append(m.deletedARNs, aws.ToString(params.ResourceShareArn))
	return &ram.DeleteResourceShareOutput{}, m.deleteErr
}

func TestResourceShareList(t *testing.T) {
	t.Parallel()

	now := time.Now()
	oldTime := now.Add(-1 * time.Hour)
	newTime := now.Add(1 * time.Hour)

	shares := []types.ResourceShare{
		{
			Name:             aws.String("share-1"),
			ResourceShareArn: aws.String("arn:aws:ram:us-east-1:123456789012:resource-share/1"),
			CreationTime:     aws.Time(oldTime),
			Tags: []types.Tag{
				{Key: aws.String("env"), Value: aws.String("prod")},
			},
		},
		{
			Name:             aws.String("share-2"),
			ResourceShareArn: aws.String("arn:aws:ram:us-east-1:123456789012:resource-share/2"),
			CreationTime:     aws.Time(newTime),
			Tags: []types.Tag{
				{Key: aws.String("env"), Value: aws.String("dev")},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected: []string{
				"arn:aws:ram:us-east-1:123456789012:resource-share/1",
				"arn:aws:ram:us-east-1:123456789012:resource-share/2",
			},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("share-1")}},
				},
			},
			expected: []string{"arn:aws:ram:us-east-1:123456789012:resource-share/2"},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("prod")}},
				},
			},
			expected: []string{"arn:aws:ram:us-east-1:123456789012:resource-share/1"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{"arn:aws:ram:us-east-1:123456789012:resource-share/1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := &mockRAMResourceShareClient{
				getOutputs: []ram.GetResourceSharesOutput{{ResourceShares: shares}},
			}

			ids, err := listResourceShares(context.Background(), mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestResourceShareListPagination(t *testing.T) {
	t.Parallel()

	mock := &mockRAMResourceShareClient{
		getOutputs: []ram.GetResourceSharesOutput{
			{
				ResourceShares: []types.ResourceShare{
					{Name: aws.String("share-1"), ResourceShareArn: aws.String("arn:1")},
				},
				NextToken: aws.String("next-page"),
			},
			{
				ResourceShares: []types.ResourceShare{
					{Name: aws.String("share-2"), ResourceShareArn: aws.String("arn:2")},
				},
			},
		},
	}

	ids, err := listResourceShares(context.Background(), mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"arn:1", "arn:2"}, aws.ToStringSlice(ids))
}

func TestDeleteResourceShare(t *testing.T) {
	t.Parallel()

	mock := &mockRAMResourceShareClient{}
	err := deleteResourceShare(context.Background(), mock, aws.String("arn:test"))
	require.NoError(t, err)
	require.Equal(t, []string{"arn:test"}, mock.deletedARNs)
}

func TestDeleteResourceShareError(t *testing.T) {
	t.Parallel()

	mock := &mockRAMResourceShareClient{deleteErr: errors.New("delete failed")}
	err := deleteResourceShare(context.Background(), mock, aws.String("arn:test"))
	require.Error(t, err)
}

func TestConvertRAMTagsToMap(t *testing.T) {
	t.Parallel()

	tagMap := convertRAMTagsToMap([]types.Tag{
		{Key: aws.String("k1"), Value: aws.String("v1")},
		{Key: nil, Value: aws.String("v2")},
		{Key: aws.String("k3"), Value: nil},
	})

	require.Equal(t, map[string]string{"k1": "v1"}, tagMap)
}
