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
	"github.com/stretchr/testify/require"
)

type mockedDynamoDB struct {
	DynamoDBAPI
	ListTablesOutput    dynamodb.ListTablesOutput
	DescribeTableOutput map[string]dynamodb.DescribeTableOutput
	DeleteTableOutput   dynamodb.DeleteTableOutput
}

func (m mockedDynamoDB) ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return &m.ListTablesOutput, nil
}

func (m mockedDynamoDB) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	output := m.DescribeTableOutput[*params.TableName]
	return &output, nil
}

func (m mockedDynamoDB) DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error) {
	return &m.DeleteTableOutput, nil
}

func TestDynamoDB_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "table1"
	testName2 := "table2"
	now := time.Now()
	ddb := DynamoDB{
		Client: mockedDynamoDB{
			ListTablesOutput: dynamodb.ListTablesOutput{
				TableNames: []string{
					testName1,
					testName2,
				},
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
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ddb.getAll(context.Background(), config.Config{
				DynamoDB: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDynamoDb_NukeAll(t *testing.T) {
	t.Parallel()
	ddb := DynamoDB{
		Client: mockedDynamoDB{
			DeleteTableOutput: dynamodb.DeleteTableOutput{},
		},
	}

	err := ddb.nukeAll([]*string{aws.String("table1"), aws.String("table2")})
	require.NoError(t, err)
}
