package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ConfigServiceRecordersAPI defines the interface for Config Service Recorder operations.
type ConfigServiceRecordersAPI interface {
	DescribeConfigurationRecorders(ctx context.Context, params *configservice.DescribeConfigurationRecordersInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigurationRecordersOutput, error)
	DeleteConfigurationRecorder(ctx context.Context, params *configservice.DeleteConfigurationRecorderInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigurationRecorderOutput, error)
}

// NewConfigServiceRecorders creates a new ConfigServiceRecorders resource using the generic resource pattern.
func NewConfigServiceRecorders() AwsResource {
	return NewAwsResource(&resource.Resource[ConfigServiceRecordersAPI]{
		ResourceTypeName: "config-recorders",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ConfigServiceRecordersAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = configservice.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ConfigServiceRecorder
		},
		Lister: listConfigServiceRecorders,
		Nuker:  resource.SimpleBatchDeleter(deleteConfigServiceRecorder),
	})
}

// listConfigServiceRecorders retrieves all Config Service Recorders that match the config filters.
func listConfigServiceRecorders(ctx context.Context, client ConfigServiceRecordersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.DescribeConfigurationRecorders(ctx, &configservice.DescribeConfigurationRecordersInput{})
	if err != nil {
		return nil, err
	}

	var recorderNames []*string
	for _, recorder := range output.ConfigurationRecorders {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: recorder.Name,
		}) {
			recorderNames = append(recorderNames, recorder.Name)
		}
	}

	return recorderNames, nil
}

// deleteConfigServiceRecorder deletes a single Config Service Recorder.
func deleteConfigServiceRecorder(ctx context.Context, client ConfigServiceRecordersAPI, name *string) error {
	_, err := client.DeleteConfigurationRecorder(ctx, &configservice.DeleteConfigurationRecorderInput{
		ConfigurationRecorderName: name,
	})
	return err
}
