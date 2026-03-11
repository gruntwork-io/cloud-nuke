package mcp

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func listResourceTypesTool() mcplib.Tool {
	return mcplib.NewTool("list_resource_types",
		mcplib.WithDescription("List all supported cloud resource types that can be inspected or nuked. Returns a JSON array of resource type name strings."),
		mcplib.WithReadOnlyHintAnnotation(true),
		mcplib.WithDestructiveHintAnnotation(false),
		mcplib.WithIdempotentHintAnnotation(true),
		mcplib.WithString("provider",
			mcplib.Description("Cloud provider: 'aws' or 'gcp'."),
			mcplib.DefaultString("aws"),
		),
	)
}

func handleListResourceTypes(_ context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	provider := request.GetString("provider", "aws")

	var resourceTypes []string
	switch provider {
	case "aws":
		resourceTypes = aws.ListResourceTypes()
	case "gcp":
		resourceTypes = gcp.ListResourceTypes()
	default:
		return unsupportedProviderError(provider)
	}

	return jsonResult(resourceTypes)
}
