package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRecordersAPI interface {
	DescribeConfigurationRecorders(ctx context.Context, params *configservice.DescribeConfigurationRecordersInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigurationRecordersOutput, error)
	DeleteConfigurationRecorder(ctx context.Context, params *configservice.DeleteConfigurationRecorderInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigurationRecorderOutput, error)
}

type ConfigServiceRecorders struct {
	BaseAwsResource
	Client        ConfigServiceRecordersAPI
	Region        string
	RecorderNames []string
}

func (csr *ConfigServiceRecorders) Init(cfg aws.Config) {
	csr.Client = configservice.NewFromConfig(cfg)
}

func (csr *ConfigServiceRecorders) ResourceName() string {
	return "config-recorders"
}

func (csr *ConfigServiceRecorders) ResourceIdentifiers() []string {
	return csr.RecorderNames
}

func (csr *ConfigServiceRecorders) MaxBatchSize() int {
	return 50
}

func (csr *ConfigServiceRecorders) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ConfigServiceRecorder
}

func (csr *ConfigServiceRecorders) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := csr.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	csr.RecorderNames = aws.ToStringSlice(identifiers)
	return csr.RecorderNames, nil
}

func (csr *ConfigServiceRecorders) Nuke(configServiceRecorderNames []string) error {
	if err := csr.nukeAll(configServiceRecorderNames); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
