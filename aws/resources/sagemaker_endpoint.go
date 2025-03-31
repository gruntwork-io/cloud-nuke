package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	goerrors "github.com/gruntwork-io/go-commons/errors"
	"golang.org/x/sync/errgroup"
)

// GetAll retrieves all SageMaker Endpoints in the region.
// It applies filtering rules from the configuration and includes tag information.
func (s *SageMakerEndpoint) GetAll(c context.Context, configObj config.Config) ([]*string, error) {
	var endpointNames []*string
	paginator := sagemaker.NewListEndpointsPaginator(s.Client, &sagemaker.ListEndpointsInput{})

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

		for _, endpoint := range output.Endpoints {
			if endpoint.EndpointName != nil {
				logging.Debugf("Found SageMaker Endpoint: %s (Status: %s)", *endpoint.EndpointName, endpoint.EndpointStatus)

				// Construct the proper ARN for the endpoint
				endpointArn := fmt.Sprintf("arn:aws:sagemaker:%s:%s:endpoint/%s",
					s.Region, accountID, *endpoint.EndpointName)

				// Get tags for the endpoint
				input := &sagemaker.ListTagsInput{
					ResourceArn: aws.String(endpointArn),
				}
				tagsOutput, err := s.Client.ListTags(s.Context, input)
				if err != nil {
					logging.Debugf("Failed to get tags for endpoint %s: %s", *endpoint.EndpointName, err)
					continue
				}

				// Convert tags to map
				tagMap := util.ConvertSageMakerTagsToMap(tagsOutput.Tags)

				// Check if this endpoint has any of the specified exclusion tags
				shouldExclude := false

				// Check the newer Tags map approach
				for tag, pattern := range configObj.SageMakerEndpoint.ExcludeRule.Tags {
					if tagValue, hasTag := tagMap[tag]; hasTag {
						if pattern.RE.MatchString(tagValue) {
							shouldExclude = true
							logging.Debugf("Excluding endpoint %s due to tag '%s' with value '%s' matching pattern '%s'",
								*endpoint.EndpointName, tag, tagValue, pattern.RE.String())
							break
						}
					}
				}

				// Check the deprecated Tag/TagValue approach
				if !shouldExclude && configObj.SageMakerEndpoint.ExcludeRule.Tag != nil {
					tagName := *configObj.SageMakerEndpoint.ExcludeRule.Tag
					if tagValue, hasTag := tagMap[tagName]; hasTag {
						// If TagValue is specified, use that pattern, otherwise check if value is "true" (case-insensitive)
						if configObj.SageMakerEndpoint.ExcludeRule.TagValue != nil {
							if configObj.SageMakerEndpoint.ExcludeRule.TagValue.RE.MatchString(tagValue) {
								shouldExclude = true
								logging.Debugf("Excluding endpoint %s due to deprecated tag '%s' with value '%s' matching pattern '%s'",
									*endpoint.EndpointName, tagName, tagValue, configObj.SageMakerEndpoint.ExcludeRule.TagValue.RE.String())
							}
						} else if strings.EqualFold(tagValue, "true") {
							shouldExclude = true
							logging.Debugf("Excluding endpoint %s due to deprecated tag '%s' with default value 'true'",
								*endpoint.EndpointName, tagName)
						}
					}
				}

				// Skip this endpoint if it should be excluded
				if shouldExclude {
					continue
				}

				resourceValue := config.ResourceValue{
					Name: endpoint.EndpointName,
					Time: endpoint.CreationTime,
					Tags: tagMap,
				}

				if configObj.SageMakerEndpoint.ShouldInclude(resourceValue) {
					endpointNames = append(endpointNames, endpoint.EndpointName)
				}
			}
		}
	}
	return endpointNames, nil
}

// nukeAll deletes all provided SageMaker Endpoints in parallel.
func (s *SageMakerEndpoint) nukeAll(identifiers []string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No SageMaker Endpoints to nuke in region %s", s.Region)
		return nil
	}

	g := new(errgroup.Group)
	for _, endpointName := range identifiers {
		if endpointName == "" {
			continue
		}
		endpointName := endpointName
		g.Go(func() error {
			err := s.nukeEndpoint(endpointName)
			report.Record(report.Entry{
				Identifier:   endpointName,
				ResourceType: "SageMaker Endpoint",
				Error:        err,
			})
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return goerrors.WithStackTrace(err)
	}

	logging.Debugf("[OK] %d SageMaker Endpoint(s) deleted in %s", len(identifiers), s.Region)
	return nil
}

// nukeEndpoint deletes a single SageMaker Endpoint and waits for deletion to complete.
func (s *SageMakerEndpoint) nukeEndpoint(endpointName string) error {
	logging.Debugf("Starting deletion of SageMaker Endpoint %s in region %s", endpointName, s.Region)

	_, err := s.Client.DeleteEndpoint(s.Context, &sagemaker.DeleteEndpointInput{
		EndpointName: aws.String(endpointName),
	})
	if err != nil {
		return err
	}

	return nil
}
