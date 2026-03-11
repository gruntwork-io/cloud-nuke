package mcp

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func inspectResourcesTool() mcplib.Tool {
	opts := append([]mcplib.ToolOption{
		mcplib.WithDescription("Read-only scan to discover cloud resources without deleting them. Returns structured JSON with resources found, summary counts, and any errors. Supports both AWS and GCP providers."),
		mcplib.WithReadOnlyHintAnnotation(true),
		mcplib.WithDestructiveHintAnnotation(false),
		mcplib.WithIdempotentHintAnnotation(true),
	}, sharedResourceParams()...)

	return mcplib.NewTool("inspect_resources", opts...)
}

func (s *Server) handleInspectResources(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	provider := request.GetString("provider", "aws")

	s.auditLog(map[string]any{
		"action":   "inspect_resources",
		"provider": provider,
		"args":     request.GetArguments(),
	})

	switch provider {
	case "aws":
		return s.handleInspectAWS(ctx, request)
	case "gcp":
		return s.handleInspectGCP(ctx, request)
	default:
		return unsupportedProviderError(provider)
	}
}

func (s *Server) handleInspectAWS(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query, common, errResult := s.parseAWSParams(request)
	if errResult != nil {
		return errResult, nil
	}

	renderer, collector := newCollectorPair()
	defer collector.Complete()

	_, err := aws.GetAllResources(ctx, query, common.configObj, collector)
	if err != nil {
		return mcplib.NewToolResultError("scan failed: " + err.Error()), nil
	}

	return jsonResult(renderer.BuildInspectResult())
}

func (s *Server) handleInspectGCP(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query, common, errResult := s.parseGCPParams(request)
	if errResult != nil {
		return errResult, nil
	}

	renderer, collector := newCollectorPair()
	defer collector.Complete()

	_, err := gcp.GetAllResources(ctx, query, common.configObj, collector)
	if err != nil {
		return mcplib.NewToolResultError("scan failed: " + err.Error()), nil
	}

	return jsonResult(renderer.BuildInspectResult())
}
