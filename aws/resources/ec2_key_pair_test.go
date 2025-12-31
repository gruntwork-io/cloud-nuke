package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEC2KeyPairsClient struct {
	DescribeKeyPairsOutput ec2.DescribeKeyPairsOutput
	DeleteKeyPairOutput    ec2.DeleteKeyPairOutput
}

func (m *mockEC2KeyPairsClient) DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	return &m.DescribeKeyPairsOutput, nil
}

func (m *mockEC2KeyPairsClient) DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error) {
	return &m.DeleteKeyPairOutput, nil
}

func TestEC2KeyPairs_ResourceName(t *testing.T) {
	r := NewEC2KeyPairs()
	assert.Equal(t, "ec2-keypairs", r.ResourceName())
}

func TestEC2KeyPairs_MaxBatchSize(t *testing.T) {
	r := NewEC2KeyPairs()
	assert.Equal(t, 200, r.MaxBatchSize())
}

func TestListEC2KeyPairs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockEC2KeyPairsClient{
		DescribeKeyPairsOutput: ec2.DescribeKeyPairsOutput{
			KeyPairs: []types.KeyPairInfo{
				{KeyPairId: aws.String("key-1"), KeyName: aws.String("keypair1"), CreateTime: aws.Time(now)},
				{KeyPairId: aws.String("key-2"), KeyName: aws.String("keypair2"), CreateTime: aws.Time(now)},
			},
		},
	}

	ids, err := listEC2KeyPairs(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"key-1", "key-2"}, aws.ToStringSlice(ids))
}

func TestListEC2KeyPairs_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockEC2KeyPairsClient{
		DescribeKeyPairsOutput: ec2.DescribeKeyPairsOutput{
			KeyPairs: []types.KeyPairInfo{
				{KeyPairId: aws.String("key-1"), KeyName: aws.String("keypair1"), CreateTime: aws.Time(now)},
				{KeyPairId: aws.String("key-2"), KeyName: aws.String("skip-this"), CreateTime: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	ids, err := listEC2KeyPairs(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"key-1"}, aws.ToStringSlice(ids))
}

func TestDeleteEC2KeyPair(t *testing.T) {
	t.Parallel()

	mock := &mockEC2KeyPairsClient{}
	err := deleteEC2KeyPair(context.Background(), mock, aws.String("key-1"))
	require.NoError(t, err)
}
