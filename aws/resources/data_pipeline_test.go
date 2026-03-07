package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datapipeline"
	dptypes "github.com/aws/aws-sdk-go-v2/service/datapipeline/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDataPipelineClient struct {
	ListPipelinesPages     []datapipeline.ListPipelinesOutput
	DescribePipelinesPages []datapipeline.DescribePipelinesOutput
	listCallIndex          int
	describeCallIndex      int
}

func (m *mockDataPipelineClient) ListPipelines(ctx context.Context, params *datapipeline.ListPipelinesInput, optFns ...func(*datapipeline.Options)) (*datapipeline.ListPipelinesOutput, error) {
	if m.listCallIndex >= len(m.ListPipelinesPages) {
		return &datapipeline.ListPipelinesOutput{}, nil
	}
	output := m.ListPipelinesPages[m.listCallIndex]
	m.listCallIndex++
	return &output, nil
}

func (m *mockDataPipelineClient) DescribePipelines(ctx context.Context, params *datapipeline.DescribePipelinesInput, optFns ...func(*datapipeline.Options)) (*datapipeline.DescribePipelinesOutput, error) {
	if m.describeCallIndex >= len(m.DescribePipelinesPages) {
		return &datapipeline.DescribePipelinesOutput{}, nil
	}
	output := m.DescribePipelinesPages[m.describeCallIndex]
	m.describeCallIndex++
	return &output, nil
}

func (m *mockDataPipelineClient) DeletePipeline(ctx context.Context, params *datapipeline.DeletePipelineInput, optFns ...func(*datapipeline.Options)) (*datapipeline.DeletePipelineOutput, error) {
	return &datapipeline.DeletePipelineOutput{}, nil
}

func TestListDataPipelines(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	laterStr := now.Add(1 * time.Hour).Format(time.RFC3339)

	twoItemDescribe := []datapipeline.DescribePipelinesOutput{{
		PipelineDescriptionList: []dptypes.PipelineDescription{
			{PipelineId: aws.String("df-001"), Name: aws.String("pipeline1"), Fields: []dptypes.Field{{Key: aws.String("@creationTime"), StringValue: aws.String(nowStr)}}},
			{PipelineId: aws.String("df-002"), Name: aws.String("pipeline2"), Fields: []dptypes.Field{{Key: aws.String("@creationTime"), StringValue: aws.String(laterStr)}}},
		},
	}}

	twoItemList := []datapipeline.ListPipelinesOutput{{
		PipelineIdList: []dptypes.PipelineIdName{
			{Id: aws.String("df-001"), Name: aws.String("pipeline1")},
			{Id: aws.String("df-002"), Name: aws.String("pipeline2")},
		},
	}}

	tests := map[string]struct {
		mock      mockDataPipelineClient
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			mock:     mockDataPipelineClient{ListPipelinesPages: twoItemList, DescribePipelinesPages: twoItemDescribe},
			expected: []string{"df-001", "df-002"},
		},
		"nameExclusionFilter": {
			mock: mockDataPipelineClient{ListPipelinesPages: twoItemList, DescribePipelinesPages: twoItemDescribe},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("pipeline1")}},
				},
			},
			expected: []string{"df-002"},
		},
		"timeAfterExclusionFilter": {
			mock: mockDataPipelineClient{ListPipelinesPages: twoItemList, DescribePipelinesPages: twoItemDescribe},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{"df-001"},
		},
		"noPipelines": {
			mock: mockDataPipelineClient{
				ListPipelinesPages: []datapipeline.ListPipelinesOutput{{PipelineIdList: []dptypes.PipelineIdName{}}},
			},
			expected: []string{},
		},
		"missingCreationTime": {
			mock: mockDataPipelineClient{
				ListPipelinesPages: []datapipeline.ListPipelinesOutput{{
					PipelineIdList: []dptypes.PipelineIdName{{Id: aws.String("df-001"), Name: aws.String("pipeline1")}},
				}},
				DescribePipelinesPages: []datapipeline.DescribePipelinesOutput{{
					PipelineDescriptionList: []dptypes.PipelineDescription{
						{PipelineId: aws.String("df-001"), Name: aws.String("pipeline1"), Fields: []dptypes.Field{}},
					},
				}},
			},
			expected: []string{"df-001"},
		},
		"bareTimestamp": {
			mock: mockDataPipelineClient{
				ListPipelinesPages: []datapipeline.ListPipelinesOutput{{
					PipelineIdList: []dptypes.PipelineIdName{{Id: aws.String("df-001"), Name: aws.String("pipeline1")}},
				}},
				DescribePipelinesPages: []datapipeline.DescribePipelinesOutput{{
					PipelineDescriptionList: []dptypes.PipelineDescription{
						{PipelineId: aws.String("df-001"), Name: aws.String("pipeline1"), Fields: []dptypes.Field{{Key: aws.String("@creationTime"), StringValue: aws.String("2015-01-01T00:00:00")}}},
					},
				}},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			expected: []string{"df-001"},
		},
		"pagination": {
			mock: mockDataPipelineClient{
				ListPipelinesPages: []datapipeline.ListPipelinesOutput{
					{PipelineIdList: []dptypes.PipelineIdName{{Id: aws.String("df-001"), Name: aws.String("pipeline1")}}, HasMoreResults: true, Marker: aws.String("token1")},
					{PipelineIdList: []dptypes.PipelineIdName{{Id: aws.String("df-002"), Name: aws.String("pipeline2")}}},
				},
				DescribePipelinesPages: []datapipeline.DescribePipelinesOutput{
					{PipelineDescriptionList: []dptypes.PipelineDescription{{PipelineId: aws.String("df-001"), Name: aws.String("pipeline1"), Fields: []dptypes.Field{{Key: aws.String("@creationTime"), StringValue: aws.String(nowStr)}}}}},
					{PipelineDescriptionList: []dptypes.PipelineDescription{{PipelineId: aws.String("df-002"), Name: aws.String("pipeline2"), Fields: []dptypes.Field{{Key: aws.String("@creationTime"), StringValue: aws.String(nowStr)}}}}},
				},
			},
			expected: []string{"df-001", "df-002"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := tc.mock
			ids, err := listDataPipelines(context.Background(), &mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteDataPipeline(t *testing.T) {
	t.Parallel()
	err := deleteDataPipeline(context.Background(), &mockDataPipelineClient{}, aws.String("df-001"))
	require.NoError(t, err)
}
