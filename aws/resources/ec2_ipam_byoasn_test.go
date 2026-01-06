package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockEC2IPAMByoasnClient struct {
	DescribeIpamByoasnOutput     ec2.DescribeIpamByoasnOutput
	DisassociateIpamByoasnOutput ec2.DisassociateIpamByoasnOutput
}

func (m *mockEC2IPAMByoasnClient) DisassociateIpamByoasn(ctx context.Context, params *ec2.DisassociateIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateIpamByoasnOutput, error) {
	return &m.DisassociateIpamByoasnOutput, nil
}

func (m *mockEC2IPAMByoasnClient) DescribeIpamByoasn(ctx context.Context, params *ec2.DescribeIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamByoasnOutput, error) {
	return &m.DescribeIpamByoasnOutput, nil
}

func TestListEC2IPAMByoasns(t *testing.T) {
	t.Parallel()

	testId1 := "ipam-byoasn-0dfc56f901b2c3462"
	testId2 := "ipam-byoasn-0dfc56f901b2c3463"

	mock := &mockEC2IPAMByoasnClient{
		DescribeIpamByoasnOutput: ec2.DescribeIpamByoasnOutput{
			Byoasns: []types.Byoasn{
				{
					Asn:    aws.String(testId1),
					IpamId: aws.String("test-ipam-byoasn-id1"),
				},
				{
					Asn:    aws.String(testId2),
					IpamId: aws.String("test-ipam-byoasn-id2"),
				},
			},
		},
	}

	ids, err := listEC2IPAMByoasns(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{testId1, testId2}, aws.ToStringSlice(ids))
}

func TestDeleteEC2IPAMByoasn(t *testing.T) {
	t.Parallel()

	mock := &mockEC2IPAMByoasnClient{}
	err := deleteEC2IPAMByoasn(context.Background(), mock, aws.String("test-asn"))
	require.NoError(t, err)
}
