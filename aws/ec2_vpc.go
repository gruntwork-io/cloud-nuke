package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllVpcs(session *session.Session, region string, configObj config.Config) ([]*string, []Vpc, error) {
	svc := ec2.New(session)

	result, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			// Note: this filter omits the default since there is special
			// handling for default resources already
			{
				Name:   awsgo.String("is-default"),
				Values: awsgo.StringSlice([]string{"false"}),
			},
		},
	})
	if err != nil {
		return nil, nil, errors.WithStackTrace(err)
	}

	var ids []*string
	var vpcs []Vpc
	for _, vpc := range result.Vpcs {
		if shouldIncludeVpc(vpc, configObj) {
			ids = append(ids, vpc.VpcId)

			vpcs = append(vpcs, Vpc{
				VpcId:  *vpc.VpcId,
				Region: region,
				svc:    svc,
			})
		}
	}

	return ids, vpcs, nil
}

func shouldIncludeVpc(vpc *ec2.Vpc, configObj config.Config) bool {
	if vpc == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(vpc.VpcId),
		configObj.VPC.IncludeRule.NamesRegExp,
		configObj.VPC.ExcludeRule.NamesRegExp,
	)
}

func nukeAllVPCs(session *session.Session, vpcIds []string, vpcs []Vpc) error {
	if len(vpcIds) == 0 {
		logging.Logger.Info("No VPCs to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all VPCs")

	deletedVPCs := 0
	multiErr := new(multierror.Error)

	for _, vpc := range vpcs {
		if err := vpc.nuke(); err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedVPCs++
			logging.Logger.Infof("Deleted VPC: %s", vpc.VpcId)
		}
	}

	logging.Logger.Infof("[OK] %d VPC terminated", deletedVPCs)

	return nil
}
