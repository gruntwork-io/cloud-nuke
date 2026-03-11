package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v2"
)

// DefaultMaxResourcesPerNuke is the safe default limit for nuke operations.
const DefaultMaxResourcesPerNuke = 100

// maxConfigYAMLSize limits config_yaml input to prevent resource exhaustion during parsing.
const maxConfigYAMLSize = 256 * 1024 // 256KB

// commonParams holds parsed parameters shared across AWS and GCP handlers.
type commonParams struct {
	excludeAfter *time.Time
	includeAfter *time.Time
	configObj    config.Config
}

// parseCommonParams extracts older_than, newer_than, and config_yaml from the request.
// Returns a tool error result if any parameter is invalid.
func parseCommonParams(request mcplib.CallToolRequest) (*commonParams, *mcplib.CallToolResult) {
	excludeAfter, err := parseDuration(request.GetString("older_than", ""))
	if err != nil {
		return nil, mcplib.NewToolResultError("invalid older_than duration: " + err.Error())
	}
	includeAfter, err := parseDuration(request.GetString("newer_than", ""))
	if err != nil {
		return nil, mcplib.NewToolResultError("invalid newer_than duration: " + err.Error())
	}

	configObj, err := parseConfigYAML(request.GetString("config_yaml", ""))
	if err != nil {
		return nil, mcplib.NewToolResultError("invalid config_yaml: " + err.Error())
	}

	return &commonParams{
		excludeAfter: excludeAfter,
		includeAfter: includeAfter,
		configObj:    configObj,
	}, nil
}

// newCollectorPair creates a renderer and collector wired together.
// Callers must defer collector.Complete() after calling this.
func newCollectorPair() (*MCPRenderer, *reporting.Collector) {
	renderer := newMCPRenderer()
	collector := reporting.NewCollector()
	collector.AddRenderer(renderer)
	return renderer, collector
}

// requireStringSlice extracts a required non-empty string slice from the request.
// Returns a descriptive error if the parameter is missing, not an array, or contains non-strings.
func requireStringSlice(request mcplib.CallToolRequest, key string) ([]string, *mcplib.CallToolResult) {
	result, err := getStringSlice(request, key)
	if err != nil {
		return nil, mcplib.NewToolResultError(fmt.Sprintf("invalid %s: %s", key, err.Error()))
	}
	if len(result) == 0 {
		return nil, mcplib.NewToolResultError(fmt.Sprintf("%s is required and must be a non-empty array of strings", key))
	}
	return result, nil
}

// optionalStringSlice extracts an optional string slice from the request.
// Returns a descriptive error if the parameter is present but malformed.
func optionalStringSlice(request mcplib.CallToolRequest, key string) ([]string, *mcplib.CallToolResult) {
	result, err := getStringSlice(request, key)
	if err != nil {
		return nil, mcplib.NewToolResultError(fmt.Sprintf("invalid %s: %s", key, err.Error()))
	}
	return result, nil
}

func getStringSlice(request mcplib.CallToolRequest, key string) ([]string, error) {
	args := request.GetArguments()
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}

	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}

	result := make([]string, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s must contain only strings", key)
		}
		result = append(result, s)
	}
	return result, nil
}

func parseDuration(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	t := time.Now().Add(-d)
	return &t, nil
}

func parseConfigYAML(yamlStr string) (config.Config, error) {
	var configObj config.Config
	if yamlStr == "" {
		return configObj, nil
	}
	if len(yamlStr) > maxConfigYAMLSize {
		return configObj, fmt.Errorf("config_yaml exceeds maximum size of %d bytes", maxConfigYAMLSize)
	}
	err := yaml.UnmarshalStrict([]byte(yamlStr), &configObj)
	return configObj, err
}

// jsonResult is a convenience wrapper around mcplib.NewToolResultJSON.
func jsonResult(v any) (*mcplib.CallToolResult, error) {
	r, err := mcplib.NewToolResultJSON(v)
	if err != nil {
		return mcplib.NewToolResultError("failed to serialize result: " + err.Error()), nil
	}
	return r, nil
}

// unsupportedProviderError returns a tool error for unrecognized provider values.
func unsupportedProviderError(provider string) (*mcplib.CallToolResult, error) {
	return mcplib.NewToolResultError("unsupported provider: " + provider + ". Supported providers: 'aws', 'gcp'."), nil
}

// toSet converts a string slice to a lookup map.
func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

// validateAllowed checks that all items are in the allowed list (if non-empty).
func validateAllowed(items, allowed []string, itemLabel, listLabel string) error {
	if len(allowed) == 0 {
		return nil
	}
	allowedSet := toSet(allowed)
	for _, item := range items {
		if !allowedSet[item] {
			return fmt.Errorf("%s %q is not in the %s. Allowed: [%s]",
				itemLabel, item, listLabel, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// sharedResourceParams returns the MCP tool options common to both inspect and nuke tools.
func sharedResourceParams() []mcplib.ToolOption {
	return []mcplib.ToolOption{
		mcplib.WithString("provider",
			mcplib.Description("Cloud provider: 'aws' or 'gcp'."),
			mcplib.DefaultString("aws"),
		),
		mcplib.WithArray("regions",
			mcplib.Description("AWS regions to scan (e.g. [\"us-east-1\", \"us-west-2\"]). Required for AWS, ignored for GCP."),
			mcplib.Items(map[string]any{"type": "string"}),
		),
		mcplib.WithArray("resource_types",
			mcplib.Description("Resource types to target. Use list_resource_types to see valid values. Required for AWS, optional for GCP (if omitted, all GCP types are scanned)."),
			mcplib.Items(map[string]any{"type": "string"}),
		),
		mcplib.WithArray("exclude_resource_types",
			mcplib.Description("Resource types to exclude from scanning."),
			mcplib.Items(map[string]any{"type": "string"}),
		),
		mcplib.WithString("project_id",
			mcplib.Description("GCP project ID. Required for GCP provider."),
		),
		mcplib.WithString("older_than",
			mcplib.Description("Only include resources older than this duration (e.g. \"24h\", \"30m\", \"2h30m\"). Supported units: s, m, h."),
		),
		mcplib.WithString("newer_than",
			mcplib.Description("Only include resources newer than this duration (e.g. \"1h\"). Supported units: s, m, h."),
		),
		mcplib.WithString("config_yaml",
			mcplib.Description("Inline YAML configuration for resource filtering rules. Use validate_config to check syntax first."),
		),
	}
}
