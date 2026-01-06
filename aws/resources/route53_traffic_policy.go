package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// Route53TrafficPolicyAPI defines the interface for Route53 Traffic Policy operations.
type Route53TrafficPolicyAPI interface {
	ListTrafficPolicies(ctx context.Context, params *route53.ListTrafficPoliciesInput, optFns ...func(*route53.Options)) (*route53.ListTrafficPoliciesOutput, error)
	DeleteTrafficPolicy(ctx context.Context, params *route53.DeleteTrafficPolicyInput, optFns ...func(*route53.Options)) (*route53.DeleteTrafficPolicyOutput, error)
}

// NewRoute53TrafficPolicies creates a new Route53 Traffic Policy resource.
func NewRoute53TrafficPolicies() AwsResource {
	return NewAwsResource(&resource.Resource[Route53TrafficPolicyAPI]{
		ResourceTypeName: "route53-traffic-policy",
		BatchSize:        DefaultBatchSize,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[Route53TrafficPolicyAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = route53.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Route53TrafficPolicy
		},
		Lister: listRoute53TrafficPolicies,
		Nuker:  resource.SequentialDeleter(deleteRoute53TrafficPolicy),
	})
}

// listRoute53TrafficPolicies retrieves all Route53 Traffic Policies that match the config filters.
// Returns identifiers in format "id:version" since DeleteTrafficPolicy requires both.
func listRoute53TrafficPolicies(ctx context.Context, client Route53TrafficPolicyAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	// ListTrafficPolicies doesn't have a paginator in the SDK, so we handle pagination manually
	var trafficPolicyIdMarker *string
	for {
		result, err := client.ListTrafficPolicies(ctx, &route53.ListTrafficPoliciesInput{
			TrafficPolicyIdMarker: trafficPolicyIdMarker,
		})
		if err != nil {
			return nil, err
		}

		for _, summary := range result.TrafficPolicySummaries {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: summary.Name,
			}) {
				// Encode both ID and version in the identifier since DeleteTrafficPolicy requires both
				identifier := fmt.Sprintf("%s:%d", aws.ToString(summary.Id), aws.ToInt32(summary.LatestVersion))
				identifiers = append(identifiers, aws.String(identifier))
			}
		}

		if !result.IsTruncated {
			break
		}
		trafficPolicyIdMarker = result.TrafficPolicyIdMarker
	}

	return identifiers, nil
}

// deleteRoute53TrafficPolicy deletes a single Route53 Traffic Policy.
// Expects identifier in format "id:version".
func deleteRoute53TrafficPolicy(ctx context.Context, client Route53TrafficPolicyAPI, identifier *string) error {
	id, version, err := parseTrafficPolicyIdentifier(aws.ToString(identifier))
	if err != nil {
		return err
	}

	_, err = client.DeleteTrafficPolicy(ctx, &route53.DeleteTrafficPolicyInput{
		Id:      aws.String(id),
		Version: aws.Int32(version),
	})
	return err
}

// parseTrafficPolicyIdentifier parses an identifier in format "id:version".
func parseTrafficPolicyIdentifier(identifier string) (string, int32, error) {
	parts := strings.SplitN(identifier, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid traffic policy identifier format: %s", identifier)
	}

	version, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return "", 0, fmt.Errorf("invalid version in traffic policy identifier: %s", identifier)
	}

	return parts[0], int32(version), nil
}
