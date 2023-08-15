package resources

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/externalcreds"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pterm/pterm"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// returns only instance Ids of unprotected ec2 instances
func (ei *EC2Instances) filterOutProtectedInstances(output *ec2.DescribeInstancesOutput, configObj config.Config) ([]*string, error) {
	var filteredIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			attr, err := ei.Client.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
				Attribute:  awsgo.String("disableApiTermination"),
				InstanceId: awsgo.String(instanceID),
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if shouldIncludeInstanceId(instance, *attr.DisableApiTermination.Value, configObj) {
				filteredIds = append(filteredIds, &instanceID)
			}
		}
	}

	return filteredIds, nil
}

// Returns a formatted string of EC2 instance ids
func (ei *EC2Instances) getAll(configObj config.Config) ([]*string, error) {
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

	output, err := ei.Client.DescribeInstances(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	instanceIds, err := ei.filterOutProtectedInstances(output, configObj)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return instanceIds, nil
}

func shouldIncludeInstanceId(instance *ec2.Instance, protected bool, configObj config.Config) bool {
	if protected {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	instanceName := GetEC2ResourceNameTagValue(instance.Tags)
	return configObj.EC2.ShouldInclude(config.ResourceValue{
		Name: instanceName,
		Time: instance.LaunchTime,
	})
}

// Deletes all non protected EC2 instances
func (ei *EC2Instances) nukeAll(instanceIds []*string) error {
	if len(instanceIds) == 0 {
		logging.Logger.Debugf("No EC2 instances to nuke in region %s", ei.Region)
		return nil
	}

	logging.Logger.Debugf("Terminating all EC2 instances in region %s", ei.Region)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}

	_, err := ei.Client.TerminateInstances(params)
	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking EC2 Instance",
		}, map[string]interface{}{
			"region": ei.Region,
		})
		return errors.WithStackTrace(err)
	}

	err = ei.Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
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
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking EC2 Instance",
		}, map[string]interface{}{
			"region": ei.Region,
		})
		return errors.WithStackTrace(err)
	}

	logging.Logger.Debugf("[OK] %d instance(s) terminated in %s", len(instanceIds), ei.Region)
	return nil
}

func GetEc2ServiceClient(region string) ec2iface.EC2API {
	return ec2.New(externalcreds.Get(region))
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

func (v Vpc) nukeInternetGateway(spinner *pterm.SpinnerPrinter) error {
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
		msg := fmt.Sprintf("...detaching Internet Gateway %s", awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId))
		spinner.UpdateText(msg)
		logging.Logger.Debug(msg)
		_, err := v.svc.DetachInternetGateway(
			&ec2.DetachInternetGatewayInput{
				InternetGatewayId: igw.InternetGateways[0].InternetGatewayId,
				VpcId:             awsgo.String(v.VpcId),
			},
		)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		spinnerMsg := fmt.Sprintf("...deleting Internet Gateway %s", awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId))
		spinner.UpdateText(spinnerMsg)
		logging.Logger.Debugf(spinnerMsg)
		_, err = v.svc.DeleteInternetGateway(
			&ec2.DeleteInternetGatewayInput{
				InternetGatewayId: igw.InternetGateways[0].InternetGatewayId,
			},
		)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	} else {
		spinnerMsg := "...no Internet Gateway found"
		logging.Logger.Debug(spinnerMsg)
	}

	return nil
}

func (v Vpc) nukeSubnets(spinner *pterm.SpinnerPrinter) error {
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
			msg := fmt.Sprintf("...deleting subnet %s", awsgo.StringValue(subnet.SubnetId))
			spinner.UpdateText(msg)
			logging.Logger.Debug(msg)
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
		msg := "...no subnets found"
		logging.Logger.Debug(msg)
	}
	return nil
}

func (v Vpc) nukeRouteTables(spinner *pterm.SpinnerPrinter) error {
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

		msg := fmt.Sprintf("...deleting route table %s", awsgo.StringValue(routeTable.RouteTableId))
		spinner.UpdateText(msg)
		logging.Logger.Debug(msg)
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

func (v Vpc) nukeNacls(spinner *pterm.SpinnerPrinter) error {
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
		msg := fmt.Sprintf("...deleting Network ACL %s", awsgo.StringValue(networkACL.NetworkAclId))
		spinner.UpdateText(msg)
		logging.Logger.Debug(msg)
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

func (v Vpc) nukeSecurityGroups(spinner *pterm.SpinnerPrinter) error {
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
			msg := fmt.Sprintf("...deleting Security Group Rule %s", awsgo.StringValue(securityGroupRule.SecurityGroupRuleId))
			spinner.UpdateText(msg)
			logging.Logger.Debug(msg)
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
		msg := fmt.Sprintf("...deleting Security Group %s", awsgo.StringValue(securityGroup.GroupId))
		spinner.UpdateText(msg)
		logging.Logger.Debug(msg)
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

func (v Vpc) nukeEndpoints(spinner *pterm.SpinnerPrinter) error {
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
		msg := fmt.Sprintf("...deleting VPC endpoint %s", awsgo.StringValue(endpoint.VpcEndpointId))
		spinner.UpdateText(msg)
		logging.Logger.Debugf(msg)
	}

	if len(endpointIds) == 0 {
		msg := "...no endpoints found"
		spinner.UpdateText(msg)
		logging.Logger.Debug(msg)
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

func (v Vpc) nukeEgressOnlyGateways(spinner *pterm.SpinnerPrinter) error {
	allEgressGateways := []*string{}
	msg := "Finding Egress Only Internet Gateways to Nuke"
	spinner.UpdateText(msg)
	logging.Logger.Debug(msg)
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
	finalMsg := fmt.Sprintf("Found %d Egress Only Internet Gateways to Nuke.", len(allEgressGateways))
	spinner.UpdateText(finalMsg)
	logging.Logger.Debug(finalMsg)

	var allErrs *multierror.Error
	for _, gateway := range allEgressGateways {
		_, e := v.svc.DeleteEgressOnlyInternetGateway(&ec2.DeleteEgressOnlyInternetGatewayInput{EgressOnlyInternetGatewayId: gateway})
		allErrs = multierror.Append(allErrs, e)
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func (v Vpc) nukeNetworkInterfaces(spinner *pterm.SpinnerPrinter) error {
	allNetworkInterfaces := []*string{}
	msg := "Finding Elastic Network Interfaces to Nuke"
	spinner.UpdateText(msg)
	logging.Logger.Debug(msg)
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

	finalMsg := fmt.Sprintf("Found %d ELastic Network Interfaces to Nuke.", len(allNetworkInterfaces))
	spinner.UpdateText(finalMsg)
	logging.Logger.Debug(finalMsg)

	var allErrs *multierror.Error
	for _, netInterface := range allNetworkInterfaces {
		_, e := v.svc.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{NetworkInterfaceId: netInterface})
		allErrs = multierror.Append(allErrs, e)
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func (v Vpc) dissociateDhcpOptions(spinner *pterm.SpinnerPrinter) error {
	msg := "Dissociating DHCP Options"
	spinner.UpdateText(msg)
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

func (v Vpc) nukeVpc(spinner *pterm.SpinnerPrinter) error {
	msg := fmt.Sprintf("Deleting VPC %s", v.VpcId)
	spinner.UpdateText(msg)
	logging.Logger.Debug(msg)
	input := &ec2.DeleteVpcInput{
		VpcId: awsgo.String(v.VpcId),
	}
	_, err := v.svc.DeleteVpc(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func (v Vpc) nuke(spinner *pterm.SpinnerPrinter) error {
	logging.Logger.Debugf("Nuking VPC %s in region %s", v.VpcId, v.Region)
	spinner.UpdateText(fmt.Sprintf("Nuking VPC %s in region %s", v.VpcId, v.Region))

	err := v.nukeInternetGateway(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Internet Gateway for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeEgressOnlyGateways(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Egress Only Internet Gateways for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeEndpoints(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Endpoints for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeNetworkInterfaces(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Elastic Network Interfaces for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeSubnets(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Subnets for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeRouteTables(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Route Tables for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeNacls(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Network ACLs for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeSecurityGroups(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up Security Groups for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.dissociateDhcpOptions(spinner)
	if err != nil {
		logging.Logger.Debugf("Error cleaning up DHCP Options for VPC %s: %s", v.VpcId, err.Error())
		return err
	}

	err = v.nukeVpc(spinner)
	if err != nil {
		logging.Logger.Debugf("Error deleting VPC %s: %s ", v.VpcId, err)
		return err
	}
	return nil
}

func NukeVpcs(vpcs []Vpc) error {
	logging.Logger.Debugf("Deleting the following VPCs:%+v\n", vpcs)

	spinnerMsg := fmt.Sprintf("Nuking the following vpcs: %+v\n", vpcs)
	// Start a simple spinner to track progress reading all relevant AWS resources
	spinnerSuccess, spinnerErr := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(spinnerMsg)

	if spinnerErr != nil {
		return errors.WithStackTrace(spinnerErr)
	}

	for _, vpc := range vpcs {
		err := vpc.nuke(spinnerSuccess)
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
func GetEC2ResourceNameTagValue(tags []*ec2.Tag) *string {
	t := make(map[string]string)

	for _, v := range tags {
		t[awsgo.StringValue(v.Key)] = awsgo.StringValue(v.Value)
	}

	if name, ok := t["Name"]; ok {
		return &name
	}

	return nil
}
