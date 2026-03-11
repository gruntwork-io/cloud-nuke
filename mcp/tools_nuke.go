package mcp

import (
	"context"
	"fmt"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

const confirmNukeString = "CONFIRM_NUKE"

func nukeResourcesTool() mcplib.Tool {
	opts := append([]mcplib.ToolOption{
		mcplib.WithDescription("Delete cloud resources. DESTRUCTIVE OPERATION. dry_run defaults to true. To actually delete, set dry_run=false and provide the required confirmation string. Use inspect_resources first to preview what would be deleted. Supports both AWS and GCP providers."),
		mcplib.WithDestructiveHintAnnotation(true),
		mcplib.WithReadOnlyHintAnnotation(false),
		mcplib.WithIdempotentHintAnnotation(false),
	}, sharedResourceParams()...)

	opts = append(opts,
		mcplib.WithBoolean("dry_run",
			mcplib.Description("When true (default), only preview what would be deleted without actually deleting."),
			mcplib.DefaultBool(true),
		),
		mcplib.WithString("confirm",
			mcplib.Description("Required confirmation string when dry_run is false. The error message will tell you the expected value if incorrect."),
		),
	)

	return mcplib.NewTool("nuke_resources", opts...)
}

func (s *Server) handleNukeResources(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// Defense-in-depth: nuke tool is not registered in read-only mode,
	// but check again in case of direct handler invocation.
	if s.config.ReadOnly {
		return mcplib.NewToolResultError("server is running in read-only mode; nuke operations are disabled"), nil
	}

	// Validate dry_run / confirm before doing any work
	dryRun := true
	args := request.GetArguments()
	if v, ok := args["dry_run"]; ok {
		if b, ok := v.(bool); ok {
			dryRun = b
		}
	}

	confirm := request.GetString("confirm", "")
	if !dryRun && confirm != confirmNukeString {
		return mcplib.NewToolResultError(
			fmt.Sprintf("confirm must be %q when dry_run is false", confirmNukeString),
		), nil
	}

	provider := request.GetString("provider", "aws")

	// Audit log all nuke attempts (including dry runs)
	s.auditLog(map[string]any{
		"action":   "nuke_resources",
		"provider": provider,
		"dry_run":  dryRun,
		"args":     request.GetArguments(),
	})

	switch provider {
	case "aws":
		return s.handleNukeAWS(ctx, request, dryRun)
	case "gcp":
		return s.handleNukeGCP(ctx, request, dryRun)
	default:
		return unsupportedProviderError(provider)
	}
}

func (s *Server) handleNukeAWS(ctx context.Context, request mcplib.CallToolRequest, dryRun bool) (*mcplib.CallToolResult, error) {
	query, common, errResult := s.parseAWSParams(request)
	if errResult != nil {
		return errResult, nil
	}

	renderer, collector := newCollectorPair()
	defer collector.Complete()

	account, err := aws.GetAllResources(ctx, query, common.configObj, collector)
	if err != nil {
		return mcplib.NewToolResultError("scan failed: " + err.Error()), nil
	}

	if dryRun {
		return jsonResult(renderer.BuildNukeResult(true))
	}

	if err := s.checkMaxResources(account.TotalResourceCount()); err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	nukeErr := aws.NukeAllResources(ctx, account, query.Regions, collector)

	result := renderer.BuildNukeResult(false)
	if nukeErr != nil {
		result.Errors = append(result.Errors, GeneralErrorInfo{
			Description: "nuke operation encountered errors",
			Error:       nukeErr.Error(),
		})
		result.Summary.GeneralErrors = len(result.Errors)
	}
	return jsonResult(result)
}

func (s *Server) handleNukeGCP(ctx context.Context, request mcplib.CallToolRequest, dryRun bool) (*mcplib.CallToolResult, error) {
	projectID, common, errResult := s.parseGCPParams(request)
	if errResult != nil {
		return errResult, nil
	}

	query := &gcp.Query{ProjectID: projectID}
	if err := query.Validate(); err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	renderer, collector := newCollectorPair()
	defer collector.Complete()

	account, err := gcp.GetAllResources(ctx, query, common.configObj, collector)
	if err != nil {
		return mcplib.NewToolResultError("scan failed: " + err.Error()), nil
	}

	if dryRun {
		return jsonResult(renderer.BuildNukeResult(true))
	}

	if err := s.checkMaxResources(account.TotalResourceCount()); err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := gcp.NukeAllResources(ctx, account, query.Regions, collector); err != nil {
		return mcplib.NewToolResultError("nuke failed: " + err.Error()), nil
	}

	return jsonResult(renderer.BuildNukeResult(false))
}

// checkMaxResources returns an error if the resource count exceeds the configured limit.
// A MaxResourcesPerNuke of 0 or less is treated as the default to prevent
// accidental bypass when ServerConfig is constructed without DefaultServerConfig().
func (s *Server) checkMaxResources(count int) error {
	limit := s.config.MaxResourcesPerNuke
	if limit <= 0 {
		limit = DefaultMaxResourcesPerNuke
	}
	if count > limit {
		return fmt.Errorf("too many resources to nuke: %d exceeds max-resources-per-nuke limit of %d. Use more specific filters to reduce the scope",
			count, limit)
	}
	return nil
}
