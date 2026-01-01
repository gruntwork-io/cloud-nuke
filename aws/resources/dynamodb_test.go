package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDynamoDBClient struct {
	ListTablesOutput    dynamodb.ListTablesOutput
	DescribeTableOutput map[string]dynamodb.DescribeTableOutput
	DeleteTableOutput   dynamodb.DeleteTableOutput
}

func (m *mockDynamoDBClient) ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return &m.ListTablesOutput, nil
}

func (m *mockDynamoDBClient) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	output := m.DescribeTableOutput[*params.TableName]
	return &output, nil
}

func (m *mockDynamoDBClient) DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error) {
	return &m.DeleteTableOutput, nil
}

func TestListDynamoDBTables(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName1 := "table1"
	testName2 := "table2"

	mock := &mockDynamoDBClient{
		ListTablesOutput: dynamodb.ListTablesOutput{
			TableNames: []string{testName1, testName2},
		},
		DescribeTableOutput: map[string]dynamodb.DescribeTableOutput{
			testName1: {
				Table: &types.TableDescription{
					TableName:        aws.String(testName1),
					CreationDateTime: aws.Time(now),
				},
			},
			testName2: {
				Table: &types.TableDescription{
					TableName:        aws.String(testName2),
					CreationDateTime: aws.Time(now.Add(1)),
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
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listDynamoDBTables(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteDynamoDBTable(t *testing.T) {
	t.Parallel()

	mock := &mockDynamoDBClient{}
	err := deleteDynamoDBTable(context.Background(), mock, aws.String("test-table"))
	require.NoError(t, err)
}
