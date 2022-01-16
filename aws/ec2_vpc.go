package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func setFirstSeenVpcTag(svc *ec2.EC2, vpc ec2.Vpc, key string, value time.Time) error {
	// We set a first seen tag because an Elastic IP doesn't contain an attribute that gives us it's creation time
	_, err := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{vpc.VpcId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(key),
				Value: awsgo.String(value.Format(time.RFC3339)),
			},
		},
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func getFirstSeenVpcTag(vpc ec2.Vpc, key string) (*time.Time, error) {
	tags := vpc.Tags
	for _, tag := range tags {
		if *tag.Key == key {
			firstSeenTime, err := time.Parse(time.RFC3339, *tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return &firstSeenTime, nil
		}
	}

	return nil, nil
}

func getAllVpcs(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, []Vpc, error) {
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
		if shouldIncludeVpc(svc, vpc, excludeAfter, configObj) {
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

func shouldIncludeVpc(svc *ec2.EC2, vpc *ec2.Vpc, excludeAfter time.Time, configObj config.Config) bool {
	if vpc == nil {
		return false
	}

	firstSeenTime, err := getFirstSeenVpcTag(*vpc, firstSeenTagKey)
	if err != nil {
		// TODO: Log error
		return false
	}

	if firstSeenTime == nil {
		setFirstSeenVpcTag(svc, *vpc, firstSeenTagKey, time.Now().UTC())
		return false
	}

	if excludeAfter.Before(*firstSeenTime) {
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
