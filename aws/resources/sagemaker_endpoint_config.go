package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	goerrors "github.com/gruntwork-io/go-commons/errors"
	"golang.org/x/sync/errgroup"
)

// GetAll retrieves all SageMaker Endpoint Configurations in the region.
// It applies filtering rules from the configuration and includes tag information.
func (s *SageMakerEndpointConfig) GetAll(c context.Context, configObj config.Config) ([]*string, error) {
	var endpointConfigNames []*string
	paginator := sagemaker.NewListEndpointConfigsPaginator(s.Client, &sagemaker.ListEndpointConfigsInput{})

	// Get account ID from context
	accountID, ok := c.Value(util.AccountIdKey).(string)
	if !ok {
		return nil, goerrors.WithStackTrace(fmt.Errorf("unable to get account ID from context"))
	}

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(c)
		if err != nil {
			return nil, goerrors.WithStackTrace(err)
		}

		for _, endpointConfig := range output.EndpointConfigs {
			if endpointConfig.EndpointConfigName != nil {
				logging.Debugf("Found SageMaker Endpoint Configuration: %s", *endpointConfig.EndpointConfigName)

				// Construct the proper ARN for the endpoint config
				endpointConfigArn := fmt.Sprintf("arn:aws:sagemaker:%s:%s:endpoint-config/%s",
					s.Region, accountID, *endpointConfig.EndpointConfigName)

				// Get tags for the endpoint config
				input := &sagemaker.ListTagsInput{
					ResourceArn: aws.String(endpointConfigArn),
				}
				tagsOutput, err := s.Client.ListTags(s.Context, input)
				if err != nil {
					logging.Debugf("Failed to get tags for endpoint config %s: %s", *endpointConfig.EndpointConfigName, err)
					continue
				}

				// Convert tags to map
				tagMap := util.ConvertSageMakerTagsToMap(tagsOutput.Tags)

				// Check if this endpoint config has any of the specified exclusion tags
				shouldExclude := false

				// Check tags
				for tag, pattern := range configObj.SageMakerEndpointConfig.ExcludeRule.Tags {
					if tagValue, hasTag := tagMap[tag]; hasTag {
						if pattern.RE.MatchString(tagValue) {
							shouldExclude = true
							logging.Debugf("Excluding endpoint config %s due to tag '%s' with value '%s' matching pattern '%s'",
								*endpointConfig.EndpointConfigName, tag, tagValue, pattern.RE.String())
							break
						}
					}
				}

				// Skip this endpoint config if it should be excluded
				if shouldExclude {
					continue
				}

				resourceValue := config.ResourceValue{
					Name: endpointConfig.EndpointConfigName,
					Time: endpointConfig.CreationTime,
					Tags: tagMap,
				}

				if configObj.SageMakerEndpointConfig.ShouldInclude(resourceValue) {
					endpointConfigNames = append(endpointConfigNames, endpointConfig.EndpointConfigName)
				}
			}
		}
	}
	return endpointConfigNames, nil
}

// nukeAll deletes all provided SageMaker Endpoint Configurations in parallel.
func (s *SageMakerEndpointConfig) nukeAll(identifiers []string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No SageMaker Endpoint Configurations to nuke in region %s", s.Region)
		return nil
	}

	g := new(errgroup.Group)
	for _, endpointConfigName := range identifiers {
		if endpointConfigName == "" {
			continue
		}
		endpointConfigName := endpointConfigName
		g.Go(func() error {
			err := s.nukeEndpointConfig(endpointConfigName)
			report.Record(report.Entry{
				Identifier:   endpointConfigName,
				ResourceType: "SageMaker Endpoint Configuration",
				Error:        err,
			})
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return goerrors.WithStackTrace(err)
	}

	logging.Debugf("[OK] %d SageMaker Endpoint Configuration(s) deleted in %s", len(identifiers), s.Region)
	return nil
}

// nukeEndpointConfig deletes a single SageMaker Endpoint Configuration.
func (s *SageMakerEndpointConfig) nukeEndpointConfig(endpointConfigName string) error {
	logging.Debugf("Starting deletion of SageMaker Endpoint Configuration %s in region %s", endpointConfigName, s.Region)

	_, err := s.Client.DeleteEndpointConfig(s.Context, &sagemaker.DeleteEndpointConfigInput{
		EndpointConfigName: aws.String(endpointConfigName),
	})
	if err != nil {
		return err
	}

	return nil
}
