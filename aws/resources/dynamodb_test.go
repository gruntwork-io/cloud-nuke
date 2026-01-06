package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDynamoDBClient struct {
	ListTablesOutput    dynamodb.ListTablesOutput
	DescribeTableOutput map[string]dynamodb.DescribeTableOutput
	UpdateTableOutput   dynamodb.UpdateTableOutput
	UpdateTableError    error
	DeleteTableOutput   dynamodb.DeleteTableOutput
	DeleteTableError    error
}

func (m *mockDynamoDBClient) ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return &m.ListTablesOutput, nil
}

func (m *mockDynamoDBClient) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	output := m.DescribeTableOutput[*params.TableName]
	return &output, nil
}

func (m *mockDynamoDBClient) UpdateTable(ctx context.Context, params *dynamodb.UpdateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTableOutput, error) {
	return &m.UpdateTableOutput, m.UpdateTableError
}

func (m *mockDynamoDBClient) DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error) {
	return &m.DeleteTableOutput, m.DeleteTableError
}

func TestDynamoDB_ResourceName(t *testing.T) {
	t.Parallel()
	r := NewDynamoDB()
	assert.Equal(t, "dynamodb", r.ResourceName())
}

func TestDynamoDB_MaxBatchSize(t *testing.T) {
	t.Parallel()
	r := NewDynamoDB()
	assert.Equal(t, DefaultBatchSize, r.MaxBatchSize())
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
					CreationDateTime: aws.Time(now.Add(1 * time.Hour)),
				},
			},
		},
	}

	tests := []struct {
		name      string
		configObj config.ResourceType
		expected  []string
	}{
		{
			name:      "returns all tables when no filter",
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		{
			name: "filters by name exclusion",
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
		{
			name: "filters by time exclusion",
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			names, err := listDynamoDBTables(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteDynamoDBTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mock        *mockDynamoDBClient
		expectError bool
	}{
		{
			name: "successfully deletes table",
			mock: &mockDynamoDBClient{
				UpdateTableOutput: dynamodb.UpdateTableOutput{},
				DeleteTableOutput: dynamodb.DeleteTableOutput{},
			},
			expectError: false,
		},
		{
			name: "continues on update error but returns delete error",
			mock: &mockDynamoDBClient{
				UpdateTableError:  errors.New("update failed"),
				DeleteTableOutput: dynamodb.DeleteTableOutput{},
			},
			expectError: false,
		},
		{
			name: "returns error when delete fails",
			mock: &mockDynamoDBClient{
				UpdateTableOutput: dynamodb.UpdateTableOutput{},
				DeleteTableError:  errors.New("delete failed"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := deleteDynamoDBTable(context.Background(), tc.mock, aws.String("test-table"))
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
