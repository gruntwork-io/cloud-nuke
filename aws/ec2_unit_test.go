// These tests use GoMock and the ec2iface to provide a mock framework for testing the EC2 API
// Unlike other tests in cloud-nuke, nuking the default VPCs and security groups is not an option.
// Other tests within cloud-nuke depend on the default VPCs/SGs to function, and other projects
// may be using the same AWS account at the same time. Deleting the default VPCs would break things.
// Therefore, the default VPC/SG nuke testing is mocked as unit tests.
// To generate the EC2API mock, install https://github.com/golang/mock, then use the following:
// mockgen -source vendor/github.com/aws/aws-sdk-go/service/ec2/ec2iface/interface.go -destination aws/mocks/gomock-EC2API.go

package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/gruntwork-io/cloud-nuke/aws/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestVpcs(mockEC2 *mock_ec2iface.MockEC2API) []Vpc {
	return []Vpc{
		{
			Region: "ap-southeast-1",
			svc:    mockEC2,
		},
		{
			Region: "eu-west-3",
			svc:    mockEC2,
		},
		{
			Region: "ca-central-1",
			svc:    mockEC2,
		},
	}
}

func TestGetDefaultVpcs(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockEC2 := mock_ec2iface.NewMockEC2API(mockCtrl)

	vpcs := getTestVpcs(mockEC2)

	describeVpcsInput := getDefaultDescribeVpcsInput()
	describeVpcsOutputOne := &ec2.DescribeVpcsOutput{
		Vpcs: []*ec2.Vpc{
			{VpcId: awsgo.String(ExampleVpcId)},
		},
	}
	describeVpcsFunc := func(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
		return describeVpcsOutputOne, nil
	}
	describeVpcsOutputTwo := &ec2.DescribeVpcsOutput{
		Vpcs: []*ec2.Vpc{
			{VpcId: awsgo.String(ExampleVpcIdTwo)},
		},
	}
	describeVpcsFuncTwo := func(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
		return describeVpcsOutputTwo, nil
	}
	describeVpcsOutputThree := &ec2.DescribeVpcsOutput{Vpcs: []*ec2.Vpc{}}
	describeVpcsFuncThree := func(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
		return describeVpcsOutputThree, nil
	}
	gomock.InOrder(
		mockEC2.EXPECT().DescribeVpcs(describeVpcsInput).DoAndReturn(describeVpcsFunc),
		mockEC2.EXPECT().DescribeVpcs(describeVpcsInput).DoAndReturn(describeVpcsFuncTwo),
		mockEC2.EXPECT().DescribeVpcs(describeVpcsInput).DoAndReturn(describeVpcsFuncThree),
	)

	vpcs, err := GetDefaultVpcs(vpcs)
	require.NoError(t, err)
	assert.Len(t, vpcs, 2, "There should be two default VPCs")
}

func getTestVpcsWithIds(mockEC2 *mock_ec2iface.MockEC2API) []Vpc {
	return []Vpc{
		{
			Region: "ap-southeast-1",
			VpcId:  ExampleVpcId,
			svc:    mockEC2,
		},
		{
			Region: "eu-west-3",
			VpcId:  ExampleVpcIdTwo,
			svc:    mockEC2,
		},
		{
			Region: "ca-central-1",
			VpcId:  ExampleVpcIdThree,
			svc:    mockEC2,
		},
	}
}

func TestNukeVpcs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock_ec2iface.NewMockEC2API(mockCtrl)

	vpcs := getTestVpcsWithIds(mockEC2)
	for _, vpc := range vpcs {
		describeInternetGatewaysInput := getDescribeInternetGatewaysInput(vpc.VpcId)
		describeInternetGatewaysOutput := getDescribeInternetGatewaysOutput(ExampleInternetGatewayId)
		describeInternetGatewaysFunc := func(input *ec2.DescribeInternetGatewaysInput) (*ec2.DescribeInternetGatewaysOutput, error) {
			return describeInternetGatewaysOutput, nil
		}
		detachInternetGatewayInput := getDetachInternetGatewayInput(vpc.VpcId, ExampleInternetGatewayId)
		deleteInternetGatewayInput := getDeleteInternetGatewayInput(ExampleInternetGatewayId)

		describeSubnetsInput := getDescribeSubnetsInput(vpc.VpcId)
		describeSubnetsOutput := getDescribeSubnetsOutput([]string{ExampleSubnetId, ExampleSubnetIdTwo, ExampleSubnetIdThree})
		describeSubnetsFunc := func(input *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
			return describeSubnetsOutput, nil
		}
		deleteSubnetInputOne := getDeleteSubnetInput(ExampleSubnetId)
		deleteSubnetInputTwo := getDeleteSubnetInput(ExampleSubnetIdTwo)
		deleteSubnetInputThree := getDeleteSubnetInput(ExampleSubnetIdThree)

		describeRouteTablesInput := getDescribeRouteTablesInput(vpc.VpcId)
		describeRouteTablesOutput := getDescribeRouteTablesOutput([]string{ExampleRouteTableId})
		describeRouteTablesFunc := func(input *ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error) {
			return describeRouteTablesOutput, nil
		}
		deleteRouteTableInput := getDeleteRouteTableInput(ExampleRouteTableId)

		describeNetworkAclsInput := getDescribeNetworkAclsInput(vpc.VpcId)
		describeNetworkAclsOutput := getDescribeNetworkAclsOutput([]string{ExampleNetworkAclId})
		describeNetworkAclsFunc := func(input *ec2.DescribeNetworkAclsInput) (*ec2.DescribeNetworkAclsOutput, error) {
			return describeNetworkAclsOutput, nil
		}
		deleteNetworkAclInput := getDeleteNetworkAclInput(ExampleNetworkAclId)

		describeSecurityGroupsInput := getDescribeSecurityGroupsInput(vpc.VpcId)
		describeSecurityGroupsOutput := getDescribeSecurityGroupsOutput([]string{ExampleSecurityGroupId})
		describeSecurityGroupsFunc := func(input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
			return describeSecurityGroupsOutput, nil
		}
		deleteSecurityGroupInput := getDeleteSecurityGroupInput(ExampleSecurityGroupId)

		deleteVpcInput := getDeleteVpcInput(vpc.VpcId)

		gomock.InOrder(
			mockEC2.EXPECT().DescribeInternetGateways(describeInternetGatewaysInput).DoAndReturn(describeInternetGatewaysFunc),
			mockEC2.EXPECT().DetachInternetGateway(detachInternetGatewayInput),
			mockEC2.EXPECT().DeleteInternetGateway(deleteInternetGatewayInput),
			mockEC2.EXPECT().DescribeSubnets(describeSubnetsInput).DoAndReturn(describeSubnetsFunc),
			mockEC2.EXPECT().DeleteSubnet(deleteSubnetInputOne),
			mockEC2.EXPECT().DeleteSubnet(deleteSubnetInputTwo),
			mockEC2.EXPECT().DeleteSubnet(deleteSubnetInputThree),
			mockEC2.EXPECT().DescribeRouteTables(describeRouteTablesInput).DoAndReturn(describeRouteTablesFunc),
			mockEC2.EXPECT().DeleteRouteTable(deleteRouteTableInput),
			mockEC2.EXPECT().DescribeNetworkAcls(describeNetworkAclsInput).DoAndReturn(describeNetworkAclsFunc),
			mockEC2.EXPECT().DeleteNetworkAcl(deleteNetworkAclInput),
			mockEC2.EXPECT().DescribeSecurityGroups(describeSecurityGroupsInput).DoAndReturn(describeSecurityGroupsFunc),
			mockEC2.EXPECT().DeleteSecurityGroup(deleteSecurityGroupInput),
			mockEC2.EXPECT().DeleteVpc(deleteVpcInput),
		)
	}

	err := NukeVpcs(vpcs)
	require.NoError(t, err)
}

func TestNukeDefaultSecurityGroups(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock_ec2iface.NewMockEC2API(mockCtrl)

	regions := []string{
		"ap-southeast-1",
		"eu-west-3",
	}

	groups := []DefaultSecurityGroup{
		{
			Region:    regions[0],
			GroupName: "default",
			GroupId:   ExampleSecurityGroupId,
			svc:       mockEC2,
		},
		{
			Region:    regions[0],
			GroupName: "default",
			GroupId:   ExampleSecurityGroupIdTwo,
			svc:       mockEC2,
		},
		{
			Region:    regions[1],
			GroupName: "default",
			GroupId:   ExampleSecurityGroupIdThree,
			svc:       mockEC2,
		},
	}
	describeSecurityGroupsInput := getDescribeSecurityGroupsInputEmpty()
	describeSecurityGroupsOutputOne := getDescribeDefaultSecurityGroupsOutput(groups[0:2])
	describeSecurityGroupsFuncOne := func(input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
		return describeSecurityGroupsOutputOne, nil
	}
	describeSecurityGroupsOutputTwo := getDescribeDefaultSecurityGroupsOutput(groups[2:])
	describeSecurityGroupsFuncTwo := func(input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
		return describeSecurityGroupsOutputTwo, nil
	}

	gomock.InOrder(
		mockEC2.EXPECT().DescribeSecurityGroups(describeSecurityGroupsInput).DoAndReturn(describeSecurityGroupsFuncOne),
		mockEC2.EXPECT().DescribeSecurityGroups(describeSecurityGroupsInput).DoAndReturn(describeSecurityGroupsFuncTwo),
		mockEC2.EXPECT().RevokeSecurityGroupIngress(groups[0].getDefaultSecurityGroupIngressRule()),
		mockEC2.EXPECT().RevokeSecurityGroupEgress(groups[0].getDefaultSecurityGroupEgressRule()),
		mockEC2.EXPECT().RevokeSecurityGroupIngress(groups[1].getDefaultSecurityGroupIngressRule()),
		mockEC2.EXPECT().RevokeSecurityGroupEgress(groups[1].getDefaultSecurityGroupEgressRule()),
		mockEC2.EXPECT().RevokeSecurityGroupIngress(groups[2].getDefaultSecurityGroupIngressRule()),
		mockEC2.EXPECT().RevokeSecurityGroupEgress(groups[2].getDefaultSecurityGroupEgressRule()),
	)

	for range regions {
		_, err := DescribeDefaultSecurityGroups(mockEC2)
		require.NoError(t, err)
	}

	err := NukeDefaultSecurityGroupRules(groups)
	require.NoError(t, err)
}

// **********************************************************************************
// The test methodology below deletes default VPCs for reals which breaks other tests
// and hence is commented out in favor of the mock testing approach above
// **********************************************************************************
// func createRandomDefaultVpc(t *testing.T, region string) Vpc {
// 	svc := ec2.New(newSession(region))
// 	defaultVpc, err := getDefaultVpc(region)
// 	require.NoError(t, err)
// 	if defaultVpc == (Vpc{}) {
// 		vpc, err := svc.CreateDefaultVpc(nil)
// 		require.NoError(t, err)
// 		defaultVpc.Region = region
// 		defaultVpc.VpcId = awsgo.StringValue(vpc.Vpc.VpcId)
// 		defaultVpc.svc = svc
// 	}
// 	return defaultVpc
// }
//
// func getRandomDefaultVpcs(t *testing.T, howMany int) []Vpc {
// 	var defaultVpcs []Vpc
//
// 	for i := 0; i < howMany; i++ {
// 		region := getRandomRegion()
// 		defaultVpcs = append(defaultVpcs, createRandomDefaultVpc(t, region))
// 	}
// 	return defaultVpcs
// }
//
//
// func TestNukeDefaultVpcs(t *testing.T) {
// 	t.Parallel()
//
// 	// How many default VPCs to nuke for this test
// 	count := 3
//
// 	defaultVpcs := getRandomDefaultVpcs(t, count)
//
// 	err := NukeVpcs(defaultVpcs)
// 	require.NoError(t, err)
//
// 	for _, vpc := range defaultVpcs {
// 		input := &ec2.DescribeVpcsInput{
// 			Filters: []*ec2.Filter{
// 				{
// 					Name:   awsgo.String("vpc-id"),
// 					Values: []*string{awsgo.String(vpc.VpcId)},
// 				},
// 			},
// 		}
// 		result, err := vpc.svc.DescribeVpcs(input)
// 		require.NoError(t, err)
// 		assert.Len(t, result.Vpcs, 0)
// 	}
// }
