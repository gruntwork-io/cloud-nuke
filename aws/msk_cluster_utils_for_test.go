package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
)

type TestMSKCluster struct {
	cfg              aws.Config
	clusterArns      []string
	vpcID            string
	subnetIDs        []string
	securityGroupIds []string
}

func (msk *TestMSKCluster) terminate(ctx context.Context) error {
	ec2Client := ec2.NewFromConfig(msk.cfg)
	kafkaClient := kafka.NewFromConfig(msk.cfg)

	// Delete MSK clusters
	for _, clusterArn := range msk.clusterArns {

		clusterInfo, err := kafkaClient.DescribeClusterV2(ctx, &kafka.DescribeClusterV2Input{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			return err
		}

		// if the cluster is already been deleted as part of the tests, skip it
		if clusterInfo.ClusterInfo.State == kafkatypes.ClusterStateDeleting {
			continue
		}

		// else we can delete the cluster
		_, err = kafkaClient.DeleteCluster(ctx, &kafka.DeleteClusterInput{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			return err
		}
	}

	// Wait for MSK clusters to be deleted
	for _, clusterArn := range msk.clusterArns {
		err := waitForMSKClusterToBeDeleted(ctx, kafkaClient, clusterArn)
		if err != nil {
			return err
		}
	}

	// Wait for ENIs to be deleted
	err := waitForENIsToBeDeleted(ctx, ec2Client, msk.subnetIDs)
	if err != nil {
		return err
	}

	// Delete security group
	for _, securityGroupID := range msk.securityGroupIds {
		_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: &securityGroupID,
		})
		if err != nil {
			return err
		}
	}

	// Delete subnets
	for _, subnetID := range msk.subnetIDs {
		_, err := ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: &subnetID,
		})
		if err != nil {
			return err
		}
	}

	// Delete VPC
	_, err = ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: &msk.vpcID,
	})
	if err != nil {
		return err
	}

	return nil
}

func waitForMSKClusterToBeCreated(ctx context.Context, kafkaClient *kafka.Client, clusterArn string) error {
	for {
		cluster, err := kafkaClient.DescribeClusterV2(ctx, &kafka.DescribeClusterV2Input{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			return err
		}

		if cluster.ClusterInfo.State == kafkatypes.ClusterStateActive {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

func waitForMSKClusterToBeDeleted(ctx context.Context, kafkaClient *kafka.Client, clusterArn string) error {
	for {
		cluster, err := kafkaClient.DescribeClusterV2(ctx, &kafka.DescribeClusterV2Input{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			var nfe *kafkatypes.NotFoundException
			if errors.As(err, &nfe) {
				return nil
			}

			return err
		}

		if cluster.ClusterInfo.State != kafkatypes.ClusterStateDeleting {
			return fmt.Errorf("MSK cluster %s is not in deleting state", clusterArn)
		}

		time.Sleep(5 * time.Second)
	}
}

func waitForENIsToBeDeleted(ctx context.Context, ec2Client *ec2.Client, subnetIds []string) error {
	for {
		enis, err := ec2Client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("subnet-id"),
					Values: subnetIds,
				},
			},
		})
		if err != nil {
			return err
		}

		if len(enis.NetworkInterfaces) == 0 {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

func createTestMSKCluster(ctx context.Context, cfg aws.Config, clusterName string, numberOfClusters int) (TestMSKCluster, error) {
	// Create VPC and subnets
	mskCluster, err := createVPCAndSubnetsForMSK(ctx, cfg, clusterName)
	if err != nil {
		return TestMSKCluster{}, err
	}

	// Create MSK cluster
	svc := kafka.NewFromConfig(cfg)
	clusterArns := make([]string, 0, numberOfClusters)
	for i := 0; i < numberOfClusters; i++ {
		cluster, err := svc.CreateClusterV2(ctx, &kafka.CreateClusterV2Input{
			ClusterName: &clusterName,
			Serverless: &kafkatypes.ServerlessRequest{
				VpcConfigs: []kafkatypes.VpcConfig{
					{
						SubnetIds:        mskCluster.subnetIDs,
						SecurityGroupIds: mskCluster.securityGroupIds,
					},
				},
				ClientAuthentication: &kafkatypes.ServerlessClientAuthentication{
					Sasl: &kafkatypes.ServerlessSasl{
						Iam: &kafkatypes.Iam{
							Enabled: true,
						},
					},
				},
			},
		})
		if err != nil {
			return TestMSKCluster{}, err
		}

		clusterArns = append(clusterArns, *cluster.ClusterArn)
	}

	for _, clusterArn := range clusterArns {
		err := waitForMSKClusterToBeCreated(ctx, svc, clusterArn)
		if err != nil {
			return TestMSKCluster{}, err
		}
	}

	mskCluster.clusterArns = clusterArns

	return *mskCluster, nil
}

func createVPCAndSubnetsForMSK(ctx context.Context, cfg aws.Config, vpcName string) (*TestMSKCluster, error) {
	// Create VPC
	vpc, err := createMSKVPC(ctx, cfg, vpcName)
	if err != nil {
		return nil, err
	}

	subnets := make([]*ec2.CreateSubnetOutput, 0, 2)
	for i := 0; i < 2; i++ {
		subnet, err := createMSKSubnet(ctx, cfg, *vpc.Vpc.VpcId, vpcName, i)
		if err != nil {
			return nil, err
		}

		subnets = append(subnets, subnet)
	}

	securitygroup, err := createMSKSecurityGroup(ctx, cfg, *vpc.Vpc.VpcId, vpcName)
	if err != nil {
		return nil, err
	}

	subnetIds := make([]string, 0, 2)
	for _, subnet := range subnets {
		subnetIds = append(subnetIds, *subnet.Subnet.SubnetId)
	}

	return &TestMSKCluster{
		cfg:              cfg,
		vpcID:            *vpc.Vpc.VpcId,
		subnetIDs:        subnetIds,
		securityGroupIds: []string{*securitygroup.GroupId},
	}, nil
}

func createMSKVPC(ctx context.Context, cfg aws.Config, vpcName string) (*ec2.CreateVpcOutput, error) {
	svc := ec2.NewFromConfig(cfg)
	vpc, err := svc.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
		TagSpecifications: []ec2types.TagSpecification{{
			ResourceType: ec2types.ResourceTypeVpc,
			Tags: []ec2types.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(vpcName),
				},
			},
		}},
	})
	if err != nil {
		return nil, err
	}

	// Wait for VPC to be created
	svc.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId: vpc.Vpc.VpcId,
		EnableDnsHostnames: &ec2types.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
	})

	return vpc, nil
}

func availabilityZoneId(ctx context.Context, cfg aws.Config, i int) (string, error) {
	svc := ec2.NewFromConfig(cfg)
	region := cfg.Region

	azs, err := svc.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("region-name"),
				Values: []string{region},
			},
		},
	})
	if err != nil {
		return "", err
	}

	// if i is greater than the number of AZs, we loop back to the beginning
	if i >= len(azs.AvailabilityZones) {
		i = i % len(azs.AvailabilityZones)
	}

	return *azs.AvailabilityZones[i].ZoneId, nil
}

func createMSKSubnet(ctx context.Context, cfg aws.Config, vpcID, vpcName string, i int) (*ec2.CreateSubnetOutput, error) {
	svc := ec2.NewFromConfig(cfg)
	cidr := fmt.Sprintf("10.0.%d.0/24", i)

	azId, err := availabilityZoneId(ctx, cfg, i)
	if err != nil {
		return nil, err
	}

	subnet, err := svc.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:              &vpcID,
		CidrBlock:          &cidr,
		AvailabilityZoneId: &azId,
		TagSpecifications: []ec2types.TagSpecification{{
			ResourceType: ec2types.ResourceTypeSubnet,
			Tags: []ec2types.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(fmt.Sprintf("%s-%d", vpcName, i)),
				},
			},
		}},
	})
	if err != nil {
		return nil, err
	}

	return subnet, nil
}

func createMSKSecurityGroup(ctx context.Context, cfg aws.Config, vpcID, vpcName string) (*ec2.CreateSecurityGroupOutput, error) {
	svc := ec2.NewFromConfig(cfg)
	securityGroup, err := svc.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(vpcName),
		Description: aws.String("cloud-nuke-msk-test"),
		VpcId:       &vpcID,
		TagSpecifications: []ec2types.TagSpecification{{
			ResourceType: ec2types.ResourceTypeSecurityGroup,
			Tags: []ec2types.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(vpcName),
				},
			},
		}},
	})
	if err != nil {
		return nil, err
	}

	return securityGroup, nil
}
