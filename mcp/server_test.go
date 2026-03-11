package mcp

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"strings"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServer creates a Server with the given config and a no-op audit logger for tests.
func testServer(cfg ServerConfig) *Server {
	return &Server{
		config:      cfg,
		auditLogger: log.New(&strings.Builder{}, "", 0),
	}
}

// listTools returns registered tool names from a Server for test assertions.
func listTools(s *Server) map[string]bool {
	tools := s.mcpServer.ListTools()
	result := make(map[string]bool, len(tools))
	for name := range tools {
		result[name] = true
	}
	return result
}

// helper to extract text from tool result
func resultText(t *testing.T, result *mcplib.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, result.Content, "expected non-empty content")
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok, "expected TextContent, got %T", result.Content[0])
	return tc.Text
}

// --- list_resource_types ---

func TestListResourceTypesAWS(t *testing.T) {
	t.Parallel()

	result, err := handleListResourceTypes(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "aws"}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var types []string
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &types))
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "ec2")
	assert.Contains(t, types, "s3")
}

func TestListResourceTypesGCP(t *testing.T) {
	t.Parallel()

	result, err := handleListResourceTypes(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "gcp"}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var types []string
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &types))
	assert.Contains(t, types, "gcs-bucket")
	assert.Contains(t, types, "cloud-function")
}

func TestListResourceTypesUnsupportedProvider(t *testing.T) {
	t.Parallel()

	result, err := handleListResourceTypes(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "azure"}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "unsupported provider")
}

func TestListResourceTypesDefaultProvider(t *testing.T) {
	t.Parallel()

	result, err := handleListResourceTypes(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var types []string
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &types))
	assert.Contains(t, types, "ec2")
}

// --- validate_config ---

func TestValidateConfigValid(t *testing.T) {
	t.Parallel()

	result, err := handleValidateConfig(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"config_yaml": "EC2:\n  include:\n    names_regex:\n      - \"^test-.*\"",
		}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var res ValidateConfigResult
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &res))
	assert.True(t, res.Valid)
}

func TestValidateConfigInvalid(t *testing.T) {
	t.Parallel()

	result, err := handleValidateConfig(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"config_yaml": "totally_bogus_field: true"}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var res ValidateConfigResult
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &res))
	assert.False(t, res.Valid)
	assert.NotEmpty(t, res.Error)
}

func TestValidateConfigMissing(t *testing.T) {
	t.Parallel()

	result, err := handleValidateConfig(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var res ValidateConfigResult
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &res))
	assert.False(t, res.Valid)
	assert.Contains(t, res.Error, "config_yaml is required")
}

func TestValidateConfigGCP(t *testing.T) {
	t.Parallel()

	result, err := handleValidateConfig(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"config_yaml": "GCSBucket:\n  include:\n    names_regex:\n      - \"^test-.*\"",
		}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var res ValidateConfigResult
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &res))
	assert.True(t, res.Valid)
}

// --- Server registration ---

func TestNewServerRegistersTools(t *testing.T) {
	t.Parallel()

	s := NewServer("test", DefaultServerConfig())
	tools := listTools(s)

	assert.Contains(t, tools, "list_resource_types")
	assert.Contains(t, tools, "validate_config")
	assert.Contains(t, tools, "inspect_resources")
	assert.Contains(t, tools, "nuke_resources")
}

func TestNewServerReadOnlyOmitsNuke(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.ReadOnly = true
	s := NewServer("test", cfg)
	tools := listTools(s)

	assert.Contains(t, tools, "list_resource_types")
	assert.Contains(t, tools, "inspect_resources")
	assert.NotContains(t, tools, "nuke_resources")
}

// --- nuke_resources safety checks ---

func TestNukeReadOnlyMode(t *testing.T) {
	t.Parallel()

	s := testServer(ServerConfig{ReadOnly: true})
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"dry_run": false, "confirm": confirmNukeString,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "read-only mode")
}

func TestNukeMissingConfirm(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"dry_run": false,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "CONFIRM_NUKE")
}

func TestNukeWrongConfirm(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"dry_run": false, "confirm": "wrong",
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "CONFIRM_NUKE")
}

func TestNukeUnsupportedProvider(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "azure", "dry_run": true}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "unsupported provider")
}

func TestNukeGCPMissingProjectID(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "gcp", "dry_run": true}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "project_id is required")
}

func TestNukeGCPConfirmRequired(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"provider": "gcp", "project_id": "my-project", "dry_run": false,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "CONFIRM_NUKE")
}

func TestNukeAWSMissingRegions(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"resource_types": []any{"ec2"}, "dry_run": true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "regions is required")
}

func TestNukeAWSMissingResourceTypes(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "dry_run": true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "resource_types is required")
}

func TestNukeInvalidOlderThan(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"older_than": "not-a-duration", "dry_run": true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "invalid older_than")
}

// --- inspect_resources validation ---

func TestInspectMissingRegions(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"resource_types": []any{"ec2"}}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "regions is required")
}

func TestInspectMissingResourceTypes(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"regions": []any{"us-east-1"}}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "resource_types is required")
}

func TestInspectUnsupportedProvider(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "azure"}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "unsupported provider")
}

func TestInspectGCPMissingProjectID(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"provider": "gcp"}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "project_id is required")
}

func TestInspectInvalidOlderThan(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"older_than": "abc",
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "invalid older_than")
}

func TestInspectInvalidConfigYAML(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"config_yaml": "bad_field: true",
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "invalid config_yaml")
}

// --- Server config validation ---

func TestValidateRegions(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedRegions = []string{"us-east-1", "us-west-2"}
	s := testServer(cfg)

	assert.NoError(t, s.validateRegions([]string{"us-east-1"}))

	err := s.validateRegions([]string{"eu-west-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "us-east-1, us-west-2") // includes allowed list
}

func TestValidateResourceTypes(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedResourceTypes = []string{"ec2", "s3"}
	s := testServer(cfg)

	assert.NoError(t, s.validateResourceTypes([]string{"ec2"}))

	err := s.validateResourceTypes([]string{"iam-role"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ec2, s3") // includes allowed list
}

func TestValidateRegionsNoRestriction(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	assert.NoError(t, s.validateRegions([]string{"any-region"}))
}

func TestValidateProject(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedProjects = []string{"my-project", "other-project"}
	s := testServer(cfg)

	assert.NoError(t, s.validateProject("my-project"))

	err := s.validateProject("forbidden-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "my-project, other-project")
}

func TestValidateProjectNoRestriction(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	assert.NoError(t, s.validateProject("any-project"))
}

func TestGCPNukeBlockedByAllowedProjects(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedProjects = []string{"safe-project"}
	s := testServer(cfg)

	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"provider": "gcp", "project_id": "evil-project", "dry_run": true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not in the allowed projects list")
}

func TestGCPInspectBlockedByAllowedProjects(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedProjects = []string{"safe-project"}
	s := testServer(cfg)

	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"provider": "gcp", "project_id": "evil-project",
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not in the allowed projects list")
}

// --- MCPRenderer ---

func TestMCPRendererInspect(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-123", Nukable: true})
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-456", Nukable: false, Reason: "protected"})
	r.OnEvent(reporting.GeneralError{ResourceType: "s3", Description: "access denied", Error: "forbidden"})

	result := r.BuildInspectResult()
	assert.Equal(t, 2, result.Summary.TotalResources)
	assert.Equal(t, 1, result.Summary.Nukable)
	assert.Equal(t, 1, result.Summary.NonNukable)
	assert.Equal(t, 1, result.Summary.GeneralErrors)
	assert.Equal(t, 2, result.Summary.ByType["ec2"])
	assert.Equal(t, 2, result.Summary.ByRegion["us-east-1"])
	assert.Len(t, result.Resources, 2)
	assert.Len(t, result.Errors, 1)
}

func TestMCPRendererNuke(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-123", Nukable: true})
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-123", Success: true})

	result := r.BuildNukeResult(false)
	assert.False(t, result.DryRun)
	assert.Equal(t, 1, result.Summary.Found)
	assert.Equal(t, 1, result.Summary.Deleted)
	assert.Equal(t, 0, result.Summary.Failed)
	assert.Len(t, result.Deleted, 1)
	assert.Equal(t, "deleted", result.Deleted[0].Status)
}

func TestMCPRendererNukeFailed(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-123", Nukable: true})
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-123", Success: false, Error: "access denied"})

	result := r.BuildNukeResult(false)
	assert.Equal(t, 0, result.Summary.Deleted)
	assert.Equal(t, 1, result.Summary.Failed)
	assert.Equal(t, "failed", result.Deleted[0].Status)
	assert.Equal(t, "access denied", result.Deleted[0].Error)
}

func TestMCPRendererNukeDryRun(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-123", Nukable: true})

	result := r.BuildNukeResult(true)
	assert.True(t, result.DryRun)
	assert.Equal(t, 1, result.Summary.Found)
	assert.Equal(t, 0, result.Summary.Deleted)
	assert.Len(t, result.Deleted, 0)
}

func TestMCPRendererEmpty(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()

	inspect := r.BuildInspectResult()
	assert.Equal(t, 0, inspect.Summary.TotalResources)
	assert.Empty(t, inspect.Resources)

	nuke := r.BuildNukeResult(true)
	assert.Equal(t, 0, nuke.Summary.Found)
	assert.Empty(t, nuke.Deleted)
}

func TestMCPRendererGCPRegion(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()
	r.OnEvent(reporting.ResourceFound{ResourceType: "gcs-bucket", Region: "global", Identifier: "my-bucket", Nukable: true})

	result := r.BuildInspectResult()
	assert.Equal(t, 1, result.Summary.TotalResources)
	assert.Equal(t, 1, result.Summary.ByRegion["global"])
	assert.Equal(t, "gcs-bucket", result.Resources[0].ResourceType)
}

func TestMCPRendererIgnoresProgressEvents(t *testing.T) {
	t.Parallel()

	r := newMCPRenderer()
	// These events should be silently ignored without panicking
	r.OnEvent(reporting.ScanProgress{ResourceType: "ec2", Region: "us-east-1"})
	r.OnEvent(reporting.ScanStarted{Regions: []string{"us-east-1"}})
	r.OnEvent(reporting.ScanComplete{})
	r.OnEvent(reporting.NukeStarted{Total: 5})
	r.OnEvent(reporting.NukeProgress{ResourceType: "ec2", Region: "us-east-1", BatchSize: 3})
	r.OnEvent(reporting.NukeComplete{})
	r.OnEvent(reporting.Complete{})

	result := r.BuildInspectResult()
	assert.Equal(t, 0, result.Summary.TotalResources)
}

// --- Helper functions ---

func TestParseCommonParamsValid(t *testing.T) {
	t.Parallel()

	p, errResult := parseCommonParams(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"older_than": "24h", "newer_than": "1h",
		}},
	})
	assert.Nil(t, errResult)
	assert.NotNil(t, p.excludeAfter)
	assert.NotNil(t, p.includeAfter)
}

func TestParseCommonParamsInvalidDuration(t *testing.T) {
	t.Parallel()

	_, errResult := parseCommonParams(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"older_than": "bad"}},
	})
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestParseCommonParamsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, errResult := parseCommonParams(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"config_yaml": "bad_field: true"}},
	})
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestCheckMaxResources(t *testing.T) {
	t.Parallel()

	s := testServer(ServerConfig{MaxResourcesPerNuke: 10})
	assert.NoError(t, s.checkMaxResources(5))
	assert.NoError(t, s.checkMaxResources(10))
	assert.Error(t, s.checkMaxResources(11))
}

func TestCheckMaxResourcesZeroUsesDefault(t *testing.T) {
	t.Parallel()

	// A zero-value MaxResourcesPerNuke should use the default (100), not disable the limit
	s := testServer(ServerConfig{MaxResourcesPerNuke: 0})
	assert.NoError(t, s.checkMaxResources(100))
	assert.Error(t, s.checkMaxResources(101))
}

func TestConfigYAMLSizeLimit(t *testing.T) {
	t.Parallel()

	// Generate a config_yaml that exceeds the size limit
	oversized := strings.Repeat("x", maxConfigYAMLSize+1)
	result, err := handleValidateConfig(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"config_yaml": oversized,
		}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var res ValidateConfigResult
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &res))
	assert.False(t, res.Valid)
	assert.Contains(t, res.Error, "exceeds maximum size")
}

func TestExcludeResourceTypesValidatedAgainstAllowlist(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedResourceTypes = []string{"ec2", "s3"}
	s := testServer(cfg)

	// exclude_resource_types should also be validated against the allowlist
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":                []any{"us-east-1"},
			"resource_types":        []any{"ec2"},
			"exclude_resource_types": []any{"iam-role"},
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not in the allowed resource types list")
}

func TestNukeAuditLog(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	s := &Server{
		config:      DefaultServerConfig(),
		auditLogger: log.New(&buf, "", 0),
	}
	// Use an unsupported provider so it fails early but after the audit log fires
	_, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"provider": "unsupported",
			"dry_run":  true,
		}},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nuke_resources")
	assert.Contains(t, buf.String(), "dry_run")
	assert.Contains(t, buf.String(), "unsupported")
}

func TestDefaultServerConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	assert.Equal(t, DefaultMaxResourcesPerNuke, cfg.MaxResourcesPerNuke)
	assert.False(t, cfg.ReadOnly)
	assert.Empty(t, cfg.AllowedRegions)
	assert.Empty(t, cfg.AllowedResourceTypes)
	assert.Empty(t, cfg.AllowedProjects)
}

// --- getStringSlice edge cases ---

func TestGetStringSliceNonArray(t *testing.T) {
	t.Parallel()

	_, err := getStringSlice(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"regions": "not-an-array"}},
	}, "regions")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be an array")
}

func TestGetStringSliceNonStringElement(t *testing.T) {
	t.Parallel()

	_, err := getStringSlice(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"regions": []any{123}}},
	}, "regions")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must contain only strings")
}

func TestGetStringSliceEmptyArray(t *testing.T) {
	t.Parallel()

	result, err := getStringSlice(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"regions": []any{}}},
	}, "regions")
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetStringSliceMissingKey(t *testing.T) {
	t.Parallel()

	result, err := getStringSlice(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{}},
	}, "regions")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// --- requireStringSlice / optionalStringSlice ---

func TestRequireStringSliceReportsTypeError(t *testing.T) {
	t.Parallel()

	// When regions is not an array, the error should be specific about what went wrong
	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":        "not-an-array",
			"resource_types": []any{"ec2"},
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "must be an array")
}

func TestOptionalStringSliceReportsTypeError(t *testing.T) {
	t.Parallel()

	// When exclude_resource_types has non-string elements, error should be reported
	s := testServer(DefaultServerConfig())
	result, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":                []any{"us-east-1"},
			"resource_types":        []any{"ec2"},
			"exclude_resource_types": []any{123},
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "must contain only strings")
}

// --- Additional nuke validation tests ---

func TestNukeInvalidNewerThan(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"newer_than": "not-valid", "dry_run": true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "invalid newer_than")
}

func TestNukeInvalidConfigYAML(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"config_yaml": "bad_field: true", "dry_run": true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "invalid config_yaml")
}

func TestNukeExcludeResourceTypesValidatedAgainstAllowlist(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.AllowedResourceTypes = []string{"ec2", "s3"}
	s := testServer(cfg)

	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":                []any{"us-east-1"},
			"resource_types":        []any{"ec2"},
			"exclude_resource_types": []any{"iam-role"},
			"dry_run":               true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not in the allowed resource types list")
}

func TestNukeExcludeResourceTypesReportsTypeError(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":                []any{"us-east-1"},
			"resource_types":        []any{"ec2"},
			"exclude_resource_types": []any{42},
			"dry_run":               true,
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "must contain only strings")
}

// --- dry_run type coercion ---

func TestNukeDryRunStringValueDefaultsToTrue(t *testing.T) {
	t.Parallel()

	// If dry_run is passed as a string (not bool), it should default to true (safe)
	// and NOT require the confirmation string.
	// Use allowed-regions to force early failure after the dry_run/confirm check passes.
	cfg := DefaultServerConfig()
	cfg.AllowedRegions = []string{"us-west-2"}
	s := testServer(cfg)
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions": []any{"us-east-1"}, "resource_types": []any{"ec2"},
			"dry_run": "false", // string, not bool — should be treated as dry_run=true
		}},
	})
	require.NoError(t, err)
	// Should NOT get a "confirm required" error — dry_run should stay true
	text := resultText(t, result)
	assert.NotContains(t, text, "CONFIRM_NUKE")
	// It will fail on region validation, which proves it got past the confirm check
	assert.Contains(t, text, "not in the allowed regions list")
}

// --- parseCommonParams additional ---

func TestParseCommonParamsInvalidNewerThan(t *testing.T) {
	t.Parallel()

	_, errResult := parseCommonParams(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{"newer_than": "invalid"}},
	})
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
	assert.Contains(t, resultText(t, errResult), "invalid newer_than")
}

// --- parseAWSParams / parseGCPParams shared extraction ---

func TestParseAWSParamsValid(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	query, common, errResult := s.parseAWSParams(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":        []any{"us-east-1"},
			"resource_types": []any{"ec2", "s3"},
		}},
	})
	assert.Nil(t, errResult)
	assert.NotNil(t, query)
	assert.NotNil(t, common)
}

func TestParseGCPParamsValid(t *testing.T) {
	t.Parallel()

	s := testServer(DefaultServerConfig())
	query, common, errResult := s.parseGCPParams(mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"project_id": "my-project",
		}},
	})
	assert.Nil(t, errResult)
	assert.Equal(t, "my-project", query.ProjectID)
	assert.NotNil(t, common)
}

// --- inspect audit logging ---

func TestInspectAuditLog(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	s := &Server{
		config:      DefaultServerConfig(),
		auditLogger: log.New(&buf, "", 0),
	}
	// Use an unsupported provider so it fails early but after the audit log fires
	_, err := s.handleInspectResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"provider": "unsupported",
		}},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "inspect_resources")
	assert.Contains(t, buf.String(), "unsupported")
}

// --- dry_run absent defaults to true ---

func TestNukeDryRunAbsentDefaultsToTrue(t *testing.T) {
	t.Parallel()

	// When dry_run is completely absent from args, it should default to true (safe)
	cfg := DefaultServerConfig()
	cfg.AllowedRegions = []string{"us-west-2"}
	s := testServer(cfg)
	result, err := s.handleNukeResources(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: map[string]any{
			"regions":        []any{"us-east-1"},
			"resource_types": []any{"ec2"},
			// dry_run not set at all
		}},
	})
	require.NoError(t, err)
	// Should NOT get a "confirm required" error — dry_run should default to true
	text := resultText(t, result)
	assert.NotContains(t, text, "CONFIRM_NUKE")
	// It will fail on region validation, which proves it got past the confirm check
	assert.Contains(t, text, "not in the allowed regions list")
}

// --- checkMaxResources edge cases ---

func TestCheckMaxResourcesNegativeUsesDefault(t *testing.T) {
	t.Parallel()

	s := testServer(ServerConfig{MaxResourcesPerNuke: -5})
	assert.NoError(t, s.checkMaxResources(100))
	assert.Error(t, s.checkMaxResources(101))
}

func TestCheckMaxResourcesExactLimit(t *testing.T) {
	t.Parallel()

	s := testServer(ServerConfig{MaxResourcesPerNuke: 50})
	assert.NoError(t, s.checkMaxResources(50))
	assert.Error(t, s.checkMaxResources(51))
}

// --- validateAllowed ---

func TestValidateAllowedEmpty(t *testing.T) {
	t.Parallel()

	assert.NoError(t, validateAllowed([]string{"anything"}, nil, "item", "list"))
	assert.NoError(t, validateAllowed([]string{"anything"}, []string{}, "item", "list"))
}

func TestValidateAllowedBlocked(t *testing.T) {
	t.Parallel()

	err := validateAllowed([]string{"bad"}, []string{"good"}, "item", "allowed list")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `item "bad" is not in the allowed list`)
}

// --- tool annotations ---

func boolPtr(b bool) *bool { return &b }

func TestInspectToolAnnotations(t *testing.T) {
	t.Parallel()

	tool := inspectResourcesTool()
	require.NotNil(t, tool.Annotations)
	assert.Equal(t, boolPtr(true), tool.Annotations.ReadOnlyHint)
	assert.Equal(t, boolPtr(false), tool.Annotations.DestructiveHint)
}

func TestNukeToolAnnotations(t *testing.T) {
	t.Parallel()

	tool := nukeResourcesTool()
	require.NotNil(t, tool.Annotations)
	assert.Equal(t, boolPtr(true), tool.Annotations.DestructiveHint)
	assert.Equal(t, boolPtr(false), tool.Annotations.ReadOnlyHint)
}

func TestListToolAnnotations(t *testing.T) {
	t.Parallel()

	tool := listResourceTypesTool()
	require.NotNil(t, tool.Annotations)
	assert.Equal(t, boolPtr(true), tool.Annotations.ReadOnlyHint)
	assert.Equal(t, boolPtr(false), tool.Annotations.DestructiveHint)
}

func TestValidateConfigToolAnnotations(t *testing.T) {
	t.Parallel()

	tool := validateConfigTool()
	require.NotNil(t, tool.Annotations)
	assert.Equal(t, boolPtr(true), tool.Annotations.ReadOnlyHint)
	assert.Equal(t, boolPtr(false), tool.Annotations.DestructiveHint)
}

// --- parseDuration edge cases ---

func TestParseDurationZero(t *testing.T) {
	t.Parallel()

	result, err := parseDuration("0s")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestParseDurationEmpty(t *testing.T) {
	t.Parallel()

	result, err := parseDuration("")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// --- parseConfigYAML edge cases ---

func TestParseConfigYAMLEmpty(t *testing.T) {
	t.Parallel()

	cfg, err := parseConfigYAML("")
	assert.NoError(t, err)
	assert.Empty(t, cfg)
}

func TestParseConfigYAMLWhitespace(t *testing.T) {
	t.Parallel()

	// Whitespace-only YAML should parse as empty config (no strict fields violated)
	cfg, err := parseConfigYAML("   \n  \n")
	assert.NoError(t, err)
	assert.Empty(t, cfg)
}

// --- validateAllowed edge cases ---

func TestValidateAllowedEmptyItems(t *testing.T) {
	t.Parallel()

	// Empty items slice should always pass even with a non-empty allowlist
	assert.NoError(t, validateAllowed([]string{}, []string{"a", "b"}, "item", "list"))
}

// --- auditLog edge cases ---

func TestAuditLogMarshalFailure(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	s := &Server{
		config:      DefaultServerConfig(),
		auditLogger: log.New(&buf, "", 0),
	}
	// math.NaN cannot be marshaled to JSON
	s.auditLog(map[string]any{
		"bad": math.NaN(),
	})
	assert.Contains(t, buf.String(), "failed to marshal audit entry")
}

// --- jsonResult error path ---

func TestJsonResultMarshalFailure(t *testing.T) {
	t.Parallel()

	// Channel cannot be marshaled to JSON
	result, err := jsonResult(make(chan int))
	assert.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "failed to serialize")
}

