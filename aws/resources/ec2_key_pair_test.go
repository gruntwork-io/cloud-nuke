package resources

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedEC2KeyPairs struct {
	ec2iface.EC2API
	DescribeKeyPairsOutput ec2.DescribeKeyPairsOutput
	DeleteKeyPairOutput    ec2.DeleteKeyPairOutput
}

func (m mockedEC2KeyPairs) DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
	return &m.DescribeKeyPairsOutput, nil
}

func (m mockedEC2KeyPairs) DeleteKeyPair(input *ec2.DeleteKeyPairInput) (*ec2.DeleteKeyPairOutput, error) {
	return &m.DeleteKeyPairOutput, nil
}

func TestEC2KeyPairs_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	now := time.Now()
	testId1 := "test-keypair-id1"
	testName1 := "test-keypair1"
	testId2 := "test-keypair-id2"
	testName2 := "test-keypair2"
	k := EC2KeyPairs{
		Client: mockedEC2KeyPairs{
			DescribeKeyPairsOutput: ec2.DescribeKeyPairsOutput{
				KeyPairs: []*ec2.KeyPairInfo{
					{
						KeyName:    awsgo.String(testName1),
						KeyPairId:  awsgo.String(testId1),
						CreateTime: awsgo.Time(now),
					},
					{
						KeyName:    awsgo.String(testName2),
						KeyPairId:  awsgo.String(testId2),
						CreateTime: awsgo.Time(now.Add(1)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestEC2KeyPairs_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	h := EC2KeyPairs{
		Client: mockedEC2KeyPairs{
			DeleteKeyPairOutput: ec2.DeleteKeyPairOutput{},
		},
	}

	err := h.nukeAll([]*string{awsgo.String("test-keypair-id-1"), awsgo.String("test-keypair-id-2")})
	require.NoError(t, err)
}
