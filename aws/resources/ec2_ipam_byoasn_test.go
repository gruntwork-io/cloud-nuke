package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIPAMByoASN struct {
	EC2IPAMByoasnAPI
	DescribeIpamByoasnOutput     ec2.DescribeIpamByoasnOutput
	DisassociateIpamByoasnOutput ec2.DisassociateIpamByoasnOutput
}

func (m mockedIPAMByoASN) DisassociateIpamByoasn(ctx context.Context, params *ec2.DisassociateIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateIpamByoasnOutput, error) {
	return &m.DisassociateIpamByoasnOutput, nil
}

func (m mockedIPAMByoASN) DescribeIpamByoasn(ctx context.Context, params *ec2.DescribeIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamByoasnOutput, error) {
	return &m.DescribeIpamByoasnOutput, nil
}

func TestIPAMByoASN_GetAll(t *testing.T) {
	t.Parallel()

	var (
		testId1   = "ipam-byoasn-0dfc56f901b2c3462"
		testId2   = "ipam-byoasn-0dfc56f901b2c3463"
		testName1 = "test-ipam-byoasn-id1"
		testName2 = "test-ipam-byoasn-id2"
	)

	ipam := EC2IPAMByoasn{
		Client: mockedIPAMByoASN{
			DescribeIpamByoasnOutput: ec2.DescribeIpamByoasnOutput{
				Byoasns: []types.Byoasn{
					{
						Asn:    aws.String(testId1),
						IpamId: aws.String(testName1),
					},
					{
						Asn:    aws.String(testId2),
						IpamId: aws.String(testName2),
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := ipam.getAll(context.Background(), config.Config{
				EC2IPAM: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestIPAMByoASN_NukeAll(t *testing.T) {
	t.Parallel()

	ipam := EC2IPAMByoasn{
		Client: mockedIPAMByoASN{
			DisassociateIpamByoasnOutput: ec2.DisassociateIpamByoasnOutput{},
		},
	}

	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
