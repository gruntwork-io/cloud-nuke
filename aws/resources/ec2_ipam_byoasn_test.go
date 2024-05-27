package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIPAMByoASN struct {
	ec2iface.EC2API
	DescribeIpamByoasnOutput     ec2.DescribeIpamByoasnOutput
	DisassociateIpamByoasnOutput ec2.DisassociateIpamByoasnOutput
}

func (m mockedIPAMByoASN) DescribeIpamByoasnWithContext(_ awsgo.Context, _ *ec2.DescribeIpamByoasnInput, _ ...request.Option) (*ec2.DescribeIpamByoasnOutput, error) {
	return &m.DescribeIpamByoasnOutput, nil
}
func (m mockedIPAMByoASN) DisassociateIpamByoasnWithContext(_ awsgo.Context, _ *ec2.DisassociateIpamByoasnInput, _ ...request.Option) (*ec2.DisassociateIpamByoasnOutput, error) {
	return &m.DisassociateIpamByoasnOutput, nil
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
				Byoasns: []*ec2.Byoasn{
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(ids))
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
