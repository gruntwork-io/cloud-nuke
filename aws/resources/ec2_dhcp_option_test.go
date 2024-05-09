package resources

import (
	"context"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEC2DhcpOption struct {
	ec2iface.EC2API
	DescribeDhcpOptionsOutput ec2.DescribeDhcpOptionsOutput
	DeleteDhcpOptionsOutput   ec2.DeleteDhcpOptionsOutput
}

func (m mockedEC2DhcpOption) DescribeDhcpOptionsPagesWithContext(_ awsgo.Context, _ *ec2.DescribeDhcpOptionsInput, fn func(*ec2.DescribeDhcpOptionsOutput, bool) bool, _ ...request.Option) error {
	fn(&m.DescribeDhcpOptionsOutput, true)
	return nil
}

func (m mockedEC2DhcpOption) DeleteDhcpOptions(_ *ec2.DeleteDhcpOptionsInput) (*ec2.DeleteDhcpOptionsOutput, error) {
	return &m.DeleteDhcpOptionsOutput, nil
}

func TestEC2DhcpOption_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "test-id-1"
	h := EC2DhcpOption{
		Client: mockedEC2DhcpOption{
			DescribeDhcpOptionsOutput: ec2.DescribeDhcpOptionsOutput{
				DhcpOptions: []*ec2.DhcpOptions{
					{
						DhcpOptionsId: awsgo.String(testId1),
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
			expected:  []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := h.getAll(context.Background(), config.Config{
				EC2DedicatedHosts: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestEC2DhcpOption_NukeAll(t *testing.T) {

	t.Parallel()

	h := EC2DhcpOption{
		Client: mockedEC2DhcpOption{
			DeleteDhcpOptionsOutput: ec2.DeleteDhcpOptionsOutput{},
		},
	}

	err := h.nukeAll([]*string{awsgo.String("test-id-1"), awsgo.String("test-id-2")})
	require.NoError(t, err)
}
