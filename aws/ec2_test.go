package aws

import (
	"errors"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/gruntwork-io/cloud-nuke/aws/mocks"
	"github.com/gruntwork-io/cloud-nuke/util"
	gruntworkerrors "github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ExampleId                   = "a1b2c3d4e5f601345"
	ExampleIdTwo                = "a1b2c3d4e5f654321"
	ExampleIdThree              = "a1b2c3d4e5f632154"
	ExampleVpcId                = "vpc-" + ExampleId
	ExampleVpcIdTwo             = "vpc-" + ExampleIdTwo
	ExampleVpcIdThree           = "vpc-" + ExampleIdThree
	ExampleSubnetId             = "subnet-" + ExampleId
	ExampleSubnetIdTwo          = "subnet-" + ExampleIdTwo
	ExampleSubnetIdThree        = "subnet-" + ExampleIdThree
	ExampleRouteTableId         = "rtb-" + ExampleId
	ExampleNetworkAclId         = "acl-" + ExampleId
	ExampleSecurityGroupId      = "sg-" + ExampleId
	ExampleSecurityGroupIdTwo   = "sg-" + ExampleIdTwo
	ExampleSecurityGroupIdThree = "sg-" + ExampleIdThree
	ExampleInternetGatewayId    = "igw-" + ExampleId
)

// getAMIIdByName - Retrieves an AMI ImageId given the name of the Id. Used for
// retrieving a standard AMI across AWS regions.
func getAMIIdByName(svc *ec2.EC2, name string) (string, error) {
	imagesResult, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{awsgo.String("self"), awsgo.String("amazon")},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("name"),
				Values: []*string{awsgo.String(name)},
			},
		},
	})

	if err != nil {
		return "", gruntworkerrors.WithStackTrace(err)
	}

	return *imagesResult.Images[0].ImageId, nil
}

// runAndWaitForInstance - Given a preconstructed ec2.RunInstancesInput object,
// make the API call to run the instance and then wait for the instance to be
// up and running before returning.
func runAndWaitForInstance(svc *ec2.EC2, name string, params *ec2.RunInstancesInput) (ec2.Instance, error) {
	runResult, err := svc.RunInstances(params)
	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	if len(runResult.Instances) == 0 {
		err := errors.New("Could not create test EC2 instance")
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	err = svc.WaitUntilInstanceExists(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("instance-id"),
				Values: []*string{runResult.Instances[0].InstanceId},
			},
		},
	})

	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	// Add test tag to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String(name),
			},
		},
	})

	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	// EC2 Instance must be in a running before this function returns
	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("instance-id"),
				Values: []*string{runResult.Instances[0].InstanceId},
			},
		},
	})

	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	return *runResult.Instances[0], nil

}

func createTestEC2Instance(t *testing.T, session *session.Session, name string, protected bool) ec2.Instance {
	svc := ec2.New(session)

	imageID, err := getAMIIdByName(svc, "amzn-ami-hvm-2017.09.1.20180115-x86_64-gp2")
	if err != nil {
		assert.Fail(t, err.Error())
	}

	params := &ec2.RunInstancesInput{
		ImageId:               awsgo.String(imageID),
		InstanceType:          awsgo.String("t2.micro"),
		MinCount:              awsgo.Int64(1),
		MaxCount:              awsgo.Int64(1),
		DisableApiTermination: awsgo.Bool(protected),
	}
	instance, err := runAndWaitForInstance(svc, name, params)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	return instance
}

func removeEC2InstanceProtection(svc *ec2.EC2, instance *ec2.Instance) error {
	// make instance unprotected so it can be cleaned up
	_, err := svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: awsgo.Bool(false),
		},
		InstanceId: instance.InstanceId,
	})

	return err
}

func findEC2InstancesByNameTag(t *testing.T, session *session.Session, name string) []*string {
	output, err := ec2.New(session).DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	var instanceIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			// Retrive only IDs of instances with the unique test tag
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					if *tag.Value == name {
						instanceIds = append(instanceIds, &instanceID)
					}
				}
			}

		}
	}

	return instanceIds
}

func TestListInstances(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	instance := createTestEC2Instance(t, session, uniqueTestID, false)
	protectedInstance := createTestEC2Instance(t, session, uniqueTestID, true)
	// clean up after this test
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId, protectedInstance.InstanceId})

	instanceIds, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.NotContains(t, instanceIds, instance.InstanceId)
	assert.NotContains(t, instanceIds, protectedInstance.InstanceId)

	instanceIds, err = getAllEc2Instances(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.Contains(t, instanceIds, instance.InstanceId)
	assert.NotContains(t, instanceIds, protectedInstance.InstanceId)

	if err = removeEC2InstanceProtection(ec2.New(session), &protectedInstance); err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
}

func TestNukeInstances(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	createTestEC2Instance(t, session, uniqueTestID, false)

	instanceIds := findEC2InstancesByNameTag(t, session, uniqueTestID)

	if err := nukeAllEc2Instances(session, instanceIds); err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	instances, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour))

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	for _, instanceID := range instanceIds {
		assert.NotContains(t, instances, *instanceID)
	}
}

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

func getDefaultDescribeVpcsInput() *ec2.DescribeVpcsInput {
	return &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("isDefault"),
				Values: []*string{awsgo.String("true")},
			},
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

func getDescribeInternetGatewaysInput(vpcId string) *ec2.DescribeInternetGatewaysInput {
	return &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("attachment.vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeInternetGatewaysOutput(gatewayId string) *ec2.DescribeInternetGatewaysOutput {
	return &ec2.DescribeInternetGatewaysOutput{
		InternetGateways: []*ec2.InternetGateway{
			{InternetGatewayId: awsgo.String(gatewayId)},
		},
	}
}

func getDetachInternetGatewayInput(vpcId, gatewayId string) *ec2.DetachInternetGatewayInput {
	return &ec2.DetachInternetGatewayInput{
		InternetGatewayId: awsgo.String(gatewayId),
		VpcId:             awsgo.String(vpcId),
	}
}

func getDeleteInternetGatewayInput(gatewayId string) *ec2.DeleteInternetGatewayInput {
	return &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: awsgo.String(gatewayId),
	}
}

func getDescribeSubnetsInput(vpcId string) *ec2.DescribeSubnetsInput {
	return &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeSubnetsOutput(subnetIds []string) *ec2.DescribeSubnetsOutput {
	var subnets []*ec2.Subnet
	for _, subnetId := range subnetIds {
		subnets = append(subnets, &ec2.Subnet{SubnetId: awsgo.String(subnetId)})
	}
	return &ec2.DescribeSubnetsOutput{Subnets: subnets}
}

func getDeleteSubnetInput(subnetId string) *ec2.DeleteSubnetInput {
	return &ec2.DeleteSubnetInput{
		SubnetId: awsgo.String(subnetId),
	}
}

func getDescribeRouteTablesInput(vpcId string) *ec2.DescribeRouteTablesInput {
	return &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeRouteTablesOutput(routeTableIds []string) *ec2.DescribeRouteTablesOutput {
	var routeTables []*ec2.RouteTable
	for _, routeTableId := range routeTableIds {
		routeTables = append(routeTables, &ec2.RouteTable{RouteTableId: awsgo.String(routeTableId)})
	}
	return &ec2.DescribeRouteTablesOutput{RouteTables: routeTables}
}

func getDeleteRouteTableInput(routeTableId string) *ec2.DeleteRouteTableInput {
	return &ec2.DeleteRouteTableInput{
		RouteTableId: awsgo.String(routeTableId),
	}
}

func getDescribeNetworkAclsInput(vpcId string) *ec2.DescribeNetworkAclsInput {
	return &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("default"),
				Values: []*string{awsgo.String("false")},
			},
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeNetworkAclsOutput(networkAclIds []string) *ec2.DescribeNetworkAclsOutput {
	var networkAcls []*ec2.NetworkAcl
	for _, networkAclId := range networkAclIds {
		networkAcls = append(networkAcls, &ec2.NetworkAcl{NetworkAclId: awsgo.String(networkAclId)})
	}
	return &ec2.DescribeNetworkAclsOutput{NetworkAcls: networkAcls}
}

func getDeleteNetworkAclInput(networkAclId string) *ec2.DeleteNetworkAclInput {
	return &ec2.DeleteNetworkAclInput{
		NetworkAclId: awsgo.String(networkAclId),
	}
}

func getDescribeSecurityGroupsInput(vpcId string) *ec2.DescribeSecurityGroupsInput {
	return &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeSecurityGroupsOutput(securityGroupIds []string) *ec2.DescribeSecurityGroupsOutput {
	var securityGroups []*ec2.SecurityGroup
	for _, securityGroup := range securityGroupIds {
		securityGroups = append(securityGroups, &ec2.SecurityGroup{
			GroupId:   awsgo.String(securityGroup),
			GroupName: awsgo.String(""),
		})
	}
	return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: securityGroups}
}

func getDeleteSecurityGroupInput(securityGroupId string) *ec2.DeleteSecurityGroupInput {
	return &ec2.DeleteSecurityGroupInput{
		GroupId: awsgo.String(securityGroupId),
	}
}

func getDeleteVpcInput(vpcId string) *ec2.DeleteVpcInput {
	return &ec2.DeleteVpcInput{
		VpcId: awsgo.String(vpcId),
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

func getDescribeSecurityGroupsInputEmpty() *ec2.DescribeSecurityGroupsInput {
	return &ec2.DescribeSecurityGroupsInput{}
}

func getDescribeDefaultSecurityGroupsOutput(groups []DefaultSecurityGroup) *ec2.DescribeSecurityGroupsOutput {
	var securityGroups []*ec2.SecurityGroup
	for _, group := range groups {
		securityGroups = append(securityGroups, &ec2.SecurityGroup{
			GroupId:   awsgo.String(group.GroupId),
			GroupName: awsgo.String("default"),
		})
	}
	return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: securityGroups}
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
		_, err := DescribeSecurityGroups(mockEC2)
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
