package mcp

import (
	"encoding/json"
	"log"
	"os"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with cloud-nuke specific configuration.
type Server struct {
	mcpServer   *mcpserver.MCPServer
	config      ServerConfig
	auditLogger *log.Logger
}

// NewServer creates an MCP server with all cloud-nuke tools registered.
func NewServer(version string, cfg ServerConfig) *Server {
	s := &Server{
		config:      cfg,
		auditLogger: log.New(os.Stderr, "cloud-nuke-audit: ", log.LstdFlags),
	}

	mcpSrv := mcpserver.NewMCPServer(
		"cloud-nuke",
		version,
		mcpserver.WithToolCapabilities(false),
	)

	// Register tools
	mcpSrv.AddTool(listResourceTypesTool(), handleListResourceTypes)
	mcpSrv.AddTool(validateConfigTool(), handleValidateConfig)
	mcpSrv.AddTool(inspectResourcesTool(), s.handleInspectResources)

	if !cfg.ReadOnly {
		mcpSrv.AddTool(nukeResourcesTool(), s.handleNukeResources)
	}

	s.mcpServer = mcpSrv
	return s
}

// Serve starts the MCP server over stdio.
func (s *Server) Serve() error {
	// MCP uses stdout for the transport, so route error logs to stderr
	errLogger := log.New(os.Stderr, "cloud-nuke-mcp: ", log.LstdFlags)
	return mcpserver.ServeStdio(s.mcpServer, mcpserver.WithErrorLogger(errLogger))
}

// validateRegions checks that all requested regions are allowed by server config.
func (s *Server) validateRegions(regions []string) error {
	return validateAllowed(regions, s.config.AllowedRegions, "region", "allowed regions list")
}

// validateResourceTypes checks that all requested types are allowed by server config.
func (s *Server) validateResourceTypes(types []string) error {
	return validateAllowed(types, s.config.AllowedResourceTypes, "resource type", "allowed resource types list")
}

// validateProject checks that the GCP project is allowed by server config.
func (s *Server) validateProject(projectID string) error {
	return validateAllowed([]string{projectID}, s.config.AllowedProjects, "project", "allowed projects list")
}

// parseAWSParams extracts and validates all AWS-specific parameters from the request.
// Returns the constructed query, common params, or a tool error result.
func (s *Server) parseAWSParams(request mcplib.CallToolRequest) (*aws.Query, *commonParams, *mcplib.CallToolResult) {
	regions, errResult := requireStringSlice(request, "regions")
	if errResult != nil {
		return nil, nil, errResult
	}

	resourceTypes, errResult := requireStringSlice(request, "resource_types")
	if errResult != nil {
		return nil, nil, errResult
	}

	excludeResourceTypes, errResult := optionalStringSlice(request, "exclude_resource_types")
	if errResult != nil {
		return nil, nil, errResult
	}

	if err := s.validateRegions(regions); err != nil {
		return nil, nil, mcplib.NewToolResultError(err.Error())
	}
	if err := s.validateResourceTypes(resourceTypes); err != nil {
		return nil, nil, mcplib.NewToolResultError(err.Error())
	}
	if err := s.validateResourceTypes(excludeResourceTypes); err != nil {
		return nil, nil, mcplib.NewToolResultError(err.Error())
	}

	common, errResult := parseCommonParams(request)
	if errResult != nil {
		return nil, nil, errResult
	}

	query, err := aws.NewQuery(
		regions, nil, resourceTypes, excludeResourceTypes,
		common.excludeAfter, common.includeAfter,
		false, nil, false, false,
	)
	if err != nil {
		return nil, nil, mcplib.NewToolResultError("invalid query: " + err.Error())
	}

	return query, common, nil
}

// parseGCPParams extracts and validates all GCP-specific parameters from the request.
// Returns the constructed query, common params, or a tool error result.
func (s *Server) parseGCPParams(request mcplib.CallToolRequest) (*gcp.Query, *commonParams, *mcplib.CallToolResult) {
	projectID, err := request.RequireString("project_id")
	if err != nil || projectID == "" {
		return nil, nil, mcplib.NewToolResultError("project_id is required for GCP provider")
	}

	if err := s.validateProject(projectID); err != nil {
		return nil, nil, mcplib.NewToolResultError(err.Error())
	}

	resourceTypes, errResult := optionalStringSlice(request, "resource_types")
	if errResult != nil {
		return nil, nil, errResult
	}
	excludeResourceTypes, errResult := optionalStringSlice(request, "exclude_resource_types")
	if errResult != nil {
		return nil, nil, errResult
	}

	if err := s.validateResourceTypes(resourceTypes); err != nil {
		return nil, nil, mcplib.NewToolResultError(err.Error())
	}
	if err := s.validateResourceTypes(excludeResourceTypes); err != nil {
		return nil, nil, mcplib.NewToolResultError(err.Error())
	}

	common, errResult := parseCommonParams(request)
	if errResult != nil {
		return nil, nil, errResult
	}

	query := &gcp.Query{
		ProjectID:            projectID,
		ResourceTypes:        resourceTypes,
		ExcludeResourceTypes: excludeResourceTypes,
		ExcludeAfter:         common.excludeAfter,
		IncludeAfter:         common.includeAfter,
	}
	if err := query.Validate(); err != nil {
		return nil, nil, mcplib.NewToolResultError("invalid query: " + err.Error())
	}

	return query, common, nil
}

// auditLog writes a structured JSON audit entry to stderr.
func (s *Server) auditLog(entry map[string]any) {
	data, err := json.Marshal(entry)
	if err != nil {
		s.auditLogger.Printf("failed to marshal audit entry: %v", err)
		return
	}
	s.auditLogger.Println(string(data))
}
