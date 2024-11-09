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
	"github.com/stretchr/testify/require"
)

type mockedEC2KeyPairs struct {
	EC2KeyPairsAPI
	DeleteKeyPairOutput    ec2.DeleteKeyPairOutput
	DescribeKeyPairsOutput ec2.DescribeKeyPairsOutput
}

func (m mockedEC2KeyPairs) DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error) {
	return &m.DeleteKeyPairOutput, nil
}

func (m mockedEC2KeyPairs) DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	return &m.DescribeKeyPairsOutput, nil
}

func TestEC2KeyPairs_GetAll(t *testing.T) {
	t.Parallel()
	now := time.Now()
	testId1 := "test-keypair-id1"
	testName1 := "test-keypair1"
	testId2 := "test-keypair-id2"
	testName2 := "test-keypair2"
	k := EC2KeyPairs{
		Client: mockedEC2KeyPairs{
			DescribeKeyPairsOutput: ec2.DescribeKeyPairsOutput{
				KeyPairs: []types.KeyPairInfo{
					{
						KeyName:    aws.String(testName1),
						KeyPairId:  aws.String(testId1),
						CreateTime: aws.Time(now),
					},
					{
						KeyName:    aws.String(testName2),
						KeyPairId:  aws.String(testId2),
						CreateTime: aws.Time(now.Add(1)),
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
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := k.getAll(context.Background(), config.Config{
				EC2KeyPairs: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEC2KeyPairs_NukeAll(t *testing.T) {
	t.Parallel()
	h := EC2KeyPairs{
		Client: mockedEC2KeyPairs{
			DeleteKeyPairOutput: ec2.DeleteKeyPairOutput{},
		},
	}

	err := h.nukeAll([]*string{aws.String("test-keypair-id-1"), aws.String("test-keypair-id-2")})
	require.NoError(t, err)
}
