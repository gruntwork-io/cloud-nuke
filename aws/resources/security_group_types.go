package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecurityGroup struct {
	BaseAwsResource
	Client          ec2iface.EC2API
	Region          string
	SecurityGroups  []string
	NukeOnlyDefault bool
}

func (sg *SecurityGroup) Init(session *session.Session) {
	sg.BaseAwsResource.Init(session)
	sg.Client = ec2.New(session)
}

func (sg *SecurityGroup) ResourceName() string {
	return "security-group"
}

func (sg *SecurityGroup) ResourceIdentifiers() []string {
	return sg.SecurityGroups
}

func (sg *SecurityGroup) MaxBatchSize() int {
	return 50
}

// func (sg *SecurityGroup) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
// 	return configObj.SecurityGroup
// }

func (sg *SecurityGroup) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sg.SecurityGroups = aws.StringValueSlice(identifiers)
	return sg.SecurityGroups, nil
}

func (sg *SecurityGroup) Nuke(identifiers []string) error {
	if err := sg.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
