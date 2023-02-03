package aws

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// returns only instance Ids of unprotected ec2 instances
func filterOutProtectedInstances(svc *ec2.EC2, output *ec2.DescribeInstancesOutput, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	var filteredIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			attr, err := svc.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
				Attribute:  awsgo.String("disableApiTermination"),
				InstanceId: awsgo.String(instanceID),
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if shouldIncludeInstanceId(instance, excludeAfter, *attr.DisableApiTermination.Value, configObj) {
				filteredIds = append(filteredIds, &instanceID)
			}
		}
	}

	return filteredIds, nil
}

// Returns a formatted string of EC2 instance ids
func getAllEc2Instances(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("instance-state-name"),
				Values: []*string{
					awsgo.String("running"), awsgo.String("pending"),
					awsgo.String("stopped"), awsgo.String("stopping"),
				},
			},
		},
	}

	output, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	instanceIds, err := filterOutProtectedInstances(svc, output, excludeAfter, configObj)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return instanceIds, nil
}

// hasEC2ExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasEC2ExcludeTag(instance *ec2.Instance) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	for _, tag := range instance.Tags {
		if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
			return true
		}
	}
	return false
}

func shouldIncludeInstanceId(instance *ec2.Instance, excludeAfter time.Time, protected bool, configObj config.Config) bool {
	if instance == nil {
		return false
	}

	if excludeAfter.Before(*instance.LaunchTime) {
		return false
	}

	if protected {
		return false
	}

	if hasEC2ExcludeTag(instance) {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	instanceName, _ := GetEC2ResourceNameTagValue(instance.Tags)

	return config.ShouldInclude(
		instanceName,
		configObj.EC2.IncludeRule.NamesRegExp,
		configObj.EC2.ExcludeRule.NamesRegExp,
	)
}

// Deletes all non protected EC2 instances
func nukeAllEc2Instances(session *session.Session, instanceIds []*string) error {
	svc := ec2.New(session)

	if len(instanceIds) == 0 {
		logging.Logger.Debugf("No EC2 instances to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Terminating all EC2 instances in region %s", *session.Config.Region)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}

	_, err := svc.TerminateInstances(params)
	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	err = svc.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("instance-id"),
				Values: instanceIds,
			},
		},
	})

	for _, instanceID := range instanceIds {
		logging.Logger.Debugf("Terminated EC2 Instance: %s", *instanceID)
	}

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Logger.Debugf("[OK] %d instance(s) terminated in %s", len(instanceIds), *session.Config.Region)
	return nil
}

func GetEc2ServiceClient(region string) ec2iface.EC2API {
	return ec2.New(newSession(region))
}

type Vpc struct {
	Region string
	VpcId  string
	svc    ec2iface.EC2API
}

// NewVpcPerRegion merely assigns a service client and region to a VPC object
// The CLI calls this, but the tests don't because the tests need to use a
// mocked service client.
func NewVpcPerRegion(regions []string) []Vpc {
	var vpcs []Vpc
	for _, region := range regions {
		vpc := Vpc{
			svc:    GetEc2ServiceClient(region),
			Region: region,
		}
		vpcs = append(vpcs, vpc)
	}
	return vpcs
}

func GetDefaultVpcId(vpc Vpc) (string, error) {
	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("isDefault"),
				Values: []*string{awsgo.String("true")},
			},
		},
	}
	vpcs, err := vpc.svc.DescribeVpcs(input)
	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return "", errors.WithStackTrace(err)
	}
	if len(vpcs.Vpcs) == 1 {
		return awsgo.StringValue(vpcs.Vpcs[0].VpcId), nil
	} else if len(vpcs.Vpcs) > 1 {
		// More than one VPC in a region should never happen
		err = fmt.Errorf("Impossible - more than one default VPC found in region %s", vpc.Region)
		return "", errors.WithStackTrace(err)
	}
	// No default VPC
	return "", nil
}

// GetDefaultVpcs needs a slice of vpcs that already have service clients and regions
// assigned, either via NewVpcPerRegion() (as in the CLI) or manually (as in the mock tests)
func GetDefaultVpcs(vpcs []Vpc) ([]Vpc, error) {
	var outVpcs []Vpc
	for _, vpc := range vpcs {
		vpcId, err := GetDefaultVpcId(vpc)
		if err != nil {
			return outVpcs, errors.WithStackTrace(err)
		}
		if vpcId != "" {
			vpc.VpcId = vpcId
			outVpcs = append(outVpcs, vpc)
		}
	}
	return outVpcs, nil
}

func (v Vpc) nukeInternetGateway() error {
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("attachment.vpc-id"),
				Values: []*string{awsgo.String(v.VpcId)},
			},
		},
	}
	igw, err := v.svc.DescribeInternetGateways(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(igw.InternetGateways) == 1 {
		logging.Logger.Debugf("...detaching Internet Gateway %s", awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId))
		_, err := v.svc.DetachInternetGateway(
			&ec2.DetachInternetGatewayInput{
				InternetGatewayId: igw.InternetGateways[0].InternetGatewayId,
				VpcId:             awsgo.String(v.VpcId),
			},
		)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		logging.Logger.Debugf("...deleting Internet Gateway %s", awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId))
		_, err = v.svc.DeleteInternetGateway(
			&ec2.DeleteInternetGatewayInput{
				InternetGatewayId: igw.InternetGateways[0].InternetGatewayId,
			},
		)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	} else {
		logging.Logger.Debugf("...no Internet Gateway found")
	}

	return nil
}

func (v Vpc) nukeSubnets() error {
	subnets, _ := v.svc.DescribeSubnets(
		&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(v.VpcId)},
				},
			},
		},
	)
	if len(subnets.Subnets) > 0 {
		for _, subnet := range subnets.Subnets {
			logging.Logger.Debugf("...deleting subnet %s", awsgo.StringValue(subnet.SubnetId))
			_, err := v.svc.DeleteSubnet(
				&ec2.DeleteSubnetInput{
					SubnetId: subnet.SubnetId,
				},
			)
			if err != nil {
				return errors.WithStackTrace(err)
			}
		}
	} else {
		logging.Logger.Debugf("...no subnets found")
	}
	return nil
}

func (v Vpc) nukeRouteTables() error {
	routeTables, _ := v.svc.DescribeRouteTables(
		&ec2.DescribeRouteTablesInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(v.VpcId)},
				},
			},
		},
	)
	for _, routeTable := range routeTables.RouteTables {
		// Skip main route table
		if len(routeTable.Associations) > 0 && *routeTable.Associations[0].Main {
			continue
		}

		logging.Logger.Debugf("...deleting route table %s", awsgo.StringValue(routeTable.RouteTableId))
		_, err := v.svc.DeleteRouteTable(
			&ec2.DeleteRouteTableInput{
				RouteTableId: routeTable.RouteTableId,
			},
		)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}
	return nil
}

func (v Vpc) nukeNacls() error {
	networkACLs, _ := v.svc.DescribeNetworkAcls(
		&ec2.DescribeNetworkAclsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("default"),
					Values: []*string{awsgo.String("false")},
				},
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(v.VpcId)},
				},
			},
		},
	)
	for _, networkACL := range networkACLs.NetworkAcls {
		logging.Logger.Debugf("...deleting Network ACL %s", awsgo.StringValue(networkACL.NetworkAclId))
		_, err := v.svc.DeleteNetworkAcl(
			&ec2.DeleteNetworkAclInput{
				NetworkAclId: networkACL.NetworkAclId,
			},
		)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}
	return nil
}

func (v Vpc) nukeSecurityGroups() error {
	securityGroups, _ := v.svc.DescribeSecurityGroups(
		&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(v.VpcId)},
				},
			},
		},
	)

	for _, securityGroup := range securityGroups.SecurityGroups {
		securityGroupRules, _ := v.svc.DescribeSecurityGroupRules(
			&ec2.DescribeSecurityGroupRulesInput{
				Filters: []*ec2.Filter{
					{
						Name:   awsgo.String("group-id"),
						Values: []*string{securityGroup.GroupId},
					},
				},
			},
		)
		for _, securityGroupRule := range securityGroupRules.SecurityGroupRules {
			logging.Logger.Debugf("...deleting Security Group Rule %s", awsgo.StringValue(securityGroupRule.SecurityGroupRuleId))
			if *securityGroupRule.IsEgress {
				_, err := v.svc.RevokeSecurityGroupEgress(
					&ec2.RevokeSecurityGroupEgressInput{
						GroupId:              securityGroup.GroupId,
						SecurityGroupRuleIds: []*string{securityGroupRule.SecurityGroupRuleId},
					},
				)
				if err != nil {
					return errors.WithStackTrace(err)
				}
			} else {
				_, err := v.svc.RevokeSecurityGroupIngress(
					&ec2.RevokeSecurityGroupIngressInput{
						GroupId:              securityGroup.GroupId,
						SecurityGroupRuleIds: []*string{securityGroupRule.SecurityGroupRuleId},
					},
				)
				if err != nil {
					return errors.WithStackTrace(err)
				}
			}
		}
	}

	for _, securityGroup := range securityGroups.SecurityGroups {
		logging.Logger.Debugf("...deleting Security Group %s", awsgo.StringValue(securityGroup.GroupId))
		if *securityGroup.GroupName != "default" {
			_, err := v.svc.DeleteSecurityGroup(
				&ec2.DeleteSecurityGroupInput{
					GroupId: securityGroup.GroupId,
				},
			)
			if err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}
	return nil
}

func (v Vpc) nukeEndpoints() error {
	endpoints, _ := v.svc.DescribeVpcEndpoints(
		&ec2.DescribeVpcEndpointsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(v.VpcId)},
				},
			},
		},
	)

	var endpointIds []*string

	for _, endpoint := range endpoints.VpcEndpoints {
		endpointIds = append(endpointIds, endpoint.VpcEndpointId)
		logging.Logger.Debugf("...deleting VPC endpoint %s", awsgo.StringValue(endpoint.VpcEndpointId))
	}

	if len(endpointIds) == 0 {
		logging.Logger.Debugf("...no endpoints found")
		return nil
	}

	_, err := v.svc.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: endpointIds,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if err := waitForVPCEndpointsToBeDeleted(v); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (v Vpc) nukeEgressOnlyGateways() error {
	allEgressGateways := []*string{}
	logging.Logger.Debugf("Finding Egress Only Internet Gateways to Nuke")
	err := v.svc.DescribeEgressOnlyInternetGatewaysPages(
		&ec2.DescribeEgressOnlyInternetGatewaysInput{},
		func(page *ec2.DescribeEgressOnlyInternetGatewaysOutput, lastPage bool) bool {
			for _, gateway := range page.EgressOnlyInternetGateways {
				for _, attachment := range gateway.Attachments {
					if *attachment.VpcId == v.VpcId {
						allEgressGateways = append(allEgressGateways, gateway.EgressOnlyInternetGatewayId)
						break
					}
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return err
	}
	logging.Logger.Debugf("Found %d Egress Only Internet Gateways to Nuke.", len(allEgressGateways))

	var allErrs *multierror.Error
	for _, gateway := range allEgressGateways {
		_, e := v.svc.DeleteEgressOnlyInternetGateway(&ec2.DeleteEgressOnlyInternetGatewayInput{EgressOnlyInternetGatewayId: gateway})
		allErrs = multierror.Append(allErrs, e)
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func (v Vpc) nukeNetworkInterfaces() error {
	allNetworkInterfaces := []*string{}
	logging.Logger.Debugf("Finding Elastic Network Interfaces to Nuke")
	vpcIds := []string{v.VpcId}
	filters := []*ec2.Filter{{Name: awsgo.String("vpc-id"), Values: awsgo.StringSlice(vpcIds)}}
	err := v.svc.DescribeNetworkInterfacesPages(
		&ec2.DescribeNetworkInterfacesInput{
			Filters: filters,
		},
		func(page *ec2.DescribeNetworkInterfacesOutput, lastPage bool) bool {
			for _, netInterface := range page.NetworkInterfaces {
				allNetworkInterfaces = append(allNetworkInterfaces, netInterface.NetworkInterfaceId)
			}
			return !lastPage
		},
	)
	if err != nil {
		return err
	}
	logging.Logger.Debugf("Found %d ELastic Network Interfaces to Nuke.", len(allNetworkInterfaces))

	var allErrs *multierror.Error
	for _, netInterface := range allNetworkInterfaces {
		_, e := v.svc.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{NetworkInterfaceId: netInterface})
		allErrs = multierror.Append(allErrs, e)
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func (v Vpc) dissociateDhcpOptions() error {
	_, err := v.svc.AssociateDhcpOptions(&ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: awsgo.String("default"),
		VpcId:         awsgo.String(v.VpcId),
	})
	return err
}

func waitForVPCEndpointsToBeDeleted(v Vpc) error {
	for i := 0; i < 30; i++ {
		endpoints, err := v.svc.DescribeVpcEndpoints(
			&ec2.DescribeVpcEndpointsInput{
				Filters: []*ec2.Filter{
					{
						Name:   awsgo.String("vpc-id"),
						Values: []*string{awsgo.String(v.VpcId)},
					},
					{
						Name:   awsgo.String("vpc-endpoint-state"),
						Values: []*string{awsgo.String("deleting")},
					},
				},
			},
		)
		if err != nil {
			return err
		}

		if len(endpoints.VpcEndpoints) == 0 {
			return nil
		}

		time.Sleep(20 * time.Second)
		logging.Logger.Debug("Waiting for VPC endpoints to be deleted...")
	}

	return VPCEndpointDeleteTimeoutError{}
}

type VPCEndpointDeleteTimeoutError struct{}

func (e VPCEndpointDeleteTimeoutError) Error() string {
	return "Timed out waiting for VPC endpoints to be successfully deleted"
}

func (v Vpc) nukeVpc() error {
	logging.Logger.Debugf("...deleting VPC %s", v.VpcId)
	input := &ec2.DeleteVpcInput{
		VpcId: awsgo.String(v.VpcId),
	}
	_, err := v.svc.DeleteVpc(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func (v Vpc) nuke() error {
	logging.Logger.Debugf("Nuking VPC %s in region %s", v.VpcId, v.Region)

	err := v.nukeInternetGateway()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Internet Gateway for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeEgressOnlyGateways()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Egress Only Internet Gateways for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeNetworkInterfaces()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Elastic Network Interfaces for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeEndpoints()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Endpoints for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeSubnets()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Subnets for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeRouteTables()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Route Tables for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeNacls()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Network ACLs for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeSecurityGroups()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Security Groups for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.dissociateDhcpOptions()
	if err != nil {
		logging.Logger.Debugf("Error cleaning up DHCP Options for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeVpc()
	if err != nil {
		logging.Logger.Debugf("Error deleting VPC %s: %s ", v.VpcId, err)
		return err
	}
	return nil
}

func NukeVpcs(vpcs []Vpc) error {
	for _, vpc := range vpcs {
		err := vpc.nuke()

		// Record status of this resource
		e := report.Entry{
			Identifier:   vpc.VpcId,
			ResourceType: "VPC",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("Skipping to the next default VPC")
			continue
		}
	}
	logging.Logger.Debug("Finished nuking default VPCs in all regions")
	return nil
}

type DefaultSecurityGroup struct {
	GroupName string
	GroupId   string
	Region    string
	svc       ec2iface.EC2API
}

func DescribeDefaultSecurityGroups(svc ec2iface.EC2API) ([]string, error) {
	var groupIds []string
	securityGroups, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return []string{}, errors.WithStackTrace(err)
	}
	for _, securityGroup := range securityGroups.SecurityGroups {
		if *securityGroup.GroupName == "default" {
			groupIds = append(groupIds, awsgo.StringValue(securityGroup.GroupId))
		}
	}
	return groupIds, nil
}

func GetDefaultSecurityGroups(regions []string) ([]DefaultSecurityGroup, error) {
	var sgs []DefaultSecurityGroup
	for _, region := range regions {
		svc := GetEc2ServiceClient(region)
		groupIds, err := DescribeDefaultSecurityGroups(svc)
		if err != nil {
			return []DefaultSecurityGroup{}, errors.WithStackTrace(err)
		}
		for _, groupId := range groupIds {
			sg := DefaultSecurityGroup{
				GroupId:   groupId,
				Region:    region,
				GroupName: "default",
				svc:       svc,
			}
			sgs = append(sgs, sg)
		}
	}
	return sgs, nil
}

func (sg DefaultSecurityGroup) getDefaultSecurityGroupIngressRule() *ec2.RevokeSecurityGroupIngressInput {
	return &ec2.RevokeSecurityGroupIngressInput{
		GroupId: awsgo.String(sg.GroupId),
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol:       awsgo.String("-1"),
				FromPort:         awsgo.Int64(0),
				ToPort:           awsgo.Int64(0),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{{GroupId: awsgo.String(sg.GroupId)}},
			},
		},
	}
}

func (sg DefaultSecurityGroup) getDefaultSecurityGroupEgressRule() *ec2.RevokeSecurityGroupEgressInput {
	return &ec2.RevokeSecurityGroupEgressInput{
		GroupId: awsgo.String(sg.GroupId),
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol: awsgo.String("-1"),
				FromPort:   awsgo.Int64(0),
				ToPort:     awsgo.Int64(0),
				IpRanges:   []*ec2.IpRange{{CidrIp: awsgo.String("0.0.0.0/0")}},
			},
		},
	}
}

func (sg DefaultSecurityGroup) getDefaultSecurityGroupIPv6EgressRule() *ec2.RevokeSecurityGroupEgressInput {
	return &ec2.RevokeSecurityGroupEgressInput{
		GroupId: awsgo.String(sg.GroupId),
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol: awsgo.String("-1"),
				FromPort:   awsgo.Int64(0),
				ToPort:     awsgo.Int64(0),
				Ipv6Ranges: []*ec2.Ipv6Range{{CidrIpv6: awsgo.String("::/0")}},
			},
		},
	}
}

func (sg DefaultSecurityGroup) nuke() error {
	logging.Logger.Debugf("...revoking default rules from Security Group %s", sg.GroupId)
	if sg.GroupName == "default" {
		ingressRule := sg.getDefaultSecurityGroupIngressRule()
		_, err := sg.svc.RevokeSecurityGroupIngress(ingressRule)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "InvalidPermission.NotFound" {
				logging.Logger.Debugf("Egress rule not present (ok)")
			} else {
				return fmt.Errorf("error deleting ingress rule: %s", errors.WithStackTrace(err))
			}
		}

		egressRule := sg.getDefaultSecurityGroupEgressRule()
		_, err = sg.svc.RevokeSecurityGroupEgress(egressRule)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "InvalidPermission.NotFound" {
				logging.Logger.Debugf("Ingress rule not present (ok)")
			} else {
				return fmt.Errorf("error deleting eggress rule: %s", errors.WithStackTrace(err))
			}
		}

		ipv6EgressRule := sg.getDefaultSecurityGroupIPv6EgressRule()
		_, err = sg.svc.RevokeSecurityGroupEgress(ipv6EgressRule)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "InvalidPermission.NotFound" {
				logging.Logger.Debugf("Ingress rule not present (ok)")
			} else {
				return fmt.Errorf("error deleting eggress rule: %s", errors.WithStackTrace(err))
			}
		}
	}
	return nil
}

func NukeDefaultSecurityGroupRules(sgs []DefaultSecurityGroup) error {
	for _, sg := range sgs {
		err := sg.nuke()

		// Record status of this resource
		e := report.Entry{
			Identifier:   sg.GroupId,
			ResourceType: "Default Security Group",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("Error: %s", err)
			logging.Logger.Debugf("Skipping to the next default Security Group")
			continue
		}
	}
	logging.Logger.Debug("Finished nuking default Security Groups in all regions")
	return nil
}

// Given an slice of tags, return the value of the Name tag
func GetEC2ResourceNameTagValue(tags []*ec2.Tag) (string, error) {
	t := make(map[string]string)

	for _, v := range tags {
		t[awsgo.StringValue(v.Key)] = awsgo.StringValue(v.Value)
	}

	if name, ok := t["Name"]; ok {
		return name, nil
	}
	return "", fmt.Errorf("Resource does not have Name tag")
}
