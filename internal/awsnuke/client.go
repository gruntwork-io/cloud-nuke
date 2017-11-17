package awsnuke

import (
	"context"

	"github.com/gruntwork-io/gruntwork-cli/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var DefaultRegions = []string{
	endpoints.ApNortheast1RegionID, // Asia Pacific (Tokyo).
	endpoints.ApNortheast2RegionID, // Asia Pacific (Seoul).
	endpoints.ApSouth1RegionID,     // Asia Pacific (Mumbai).
	endpoints.ApSoutheast1RegionID, // Asia Pacific (Singapore).
	endpoints.ApSoutheast2RegionID, // Asia Pacific (Sydney).
	endpoints.CaCentral1RegionID,   // Canada (Central).
	endpoints.EuCentral1RegionID,   // EU (Frankfurt).
	endpoints.EuWest1RegionID,      // EU (Ireland).
	endpoints.EuWest2RegionID,      // EU (London).
	endpoints.SaEast1RegionID,      // South America (Sao Paulo).
	endpoints.UsEast1RegionID,      // US East (N. Virginia).
	endpoints.UsEast2RegionID,      // US East (Ohio).
	endpoints.UsWest1RegionID,      // US West (N. California).
	endpoints.UsWest2RegionID,      // US West (Oregon).
}

var DefaultAWSClientProvider = defaultAWSClientProvider{
	sess: session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewEnvCredentials(),
	})),
}

// need to move the aws service client creation out of the following methods
// and into a "ClientProvider" which will allow for this package to be unit
// tested, this was skipped in the intrest of time.

// Client is used to list and destroy resources
type Client struct {
	// Id of the AWS account to work with
	Id string

	// Regions to retrieve resources for
	Regions []string

	aws AWSClientProvider
}

// New constructs a new Client. Id is the AWS account ID that should be used.
func New(id string) *Client {
	return &Client{
		Id:      id,
		Regions: DefaultRegions,
		aws:     DefaultAWSClientProvider,
	}
}

func (c *Client) ListNonProtectedEC2Instances(ctx context.Context) ([]EC2Instance, error) {
	var allinstances []EC2Instance
	for _, region := range c.Regions {
		instances, err := c.listNonProtectedEC2InstancesInRegion(ctx, region)
		if err != nil {
			return nil, err
		}
		allinstances = append(allinstances, instances...)
	}
	return allinstances, nil
}

func (c *Client) listNonProtectedEC2InstancesInRegion(ctx context.Context, region string) ([]EC2Instance, error) {
	var instances []EC2Instance
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			filter("owner-id", c.Id),
			filter("instance-state-name", "running", "pending"),
		},
	}
	err := c.aws.ProvideEC2Client(region).
		DescribeInstancesPagesWithContext(ctx, input, func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, reservation := range page.Reservations {
				for _, instance := range reservation.Instances {
					// TODO filter out protected instances
					instances = append(instances, EC2Instance{
						Id:     aws.StringValue(instance.InstanceId),
						Region: region,
					})
				}
			}
			return true
		})
	if err != nil {
		return nil, errors.WithStackTraceAndPrefix(err, "failed to list ec2 instances")
	}
	return instances, nil
}

func (c *Client) DestroyEC2Instances(ctx context.Context, instances []EC2Instance) error {
	for _, instance := range instances {
		ec2client := c.aws.ProvideEC2Client(instance.Region)
		_, err := ec2client.TerminateInstancesWithContext(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []*string{
				aws.String(instance.Id),
			},
		})
		if err != nil {
			return errors.WithStackTraceAndPrefix(err, "falied to terminate ec2 instance, %s", instance.Id)
		}
	}
	return nil
}

// EC2Instance represents an AWS EC2 Instance
type EC2Instance struct {
	Id     string
	Region string
}

type AWSClientProvider interface {
	ProvideEC2Client(region string) *ec2.EC2
}

type defaultAWSClientProvider struct {
	sess *session.Session
}

func (p defaultAWSClientProvider) ProvideEC2Client(region string) *ec2.EC2 {
	return ec2.New(p.sess, &aws.Config{
		Region: aws.String(region),
	})
}

func filter(name, value string, v ...string) *ec2.Filter {
	values := []*string{
		aws.String(value),
	}
	for _, val := range v {
		values = append(values, aws.String(val))
	}
	return &ec2.Filter{
		Name:   aws.String(name),
		Values: values,
	}
}
