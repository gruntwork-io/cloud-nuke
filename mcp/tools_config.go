package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func validateConfigTool() mcplib.Tool {
	return mcplib.NewTool("validate_config",
		mcplib.WithDescription("Parse and validate a cloud-nuke YAML configuration string. Use this to check your config_yaml before passing it to inspect_resources or nuke_resources."),
		mcplib.WithReadOnlyHintAnnotation(true),
		mcplib.WithDestructiveHintAnnotation(false),
		mcplib.WithIdempotentHintAnnotation(true),
		mcplib.WithString("config_yaml",
			mcplib.Required(),
			mcplib.Description("The YAML configuration string to validate."),
		),
	)
}

func handleValidateConfig(_ context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	configYAML, err := request.RequireString("config_yaml")
	if err != nil {
		return jsonResult(ValidateConfigResult{
			Valid: false,
			Error: "config_yaml is required",
		})
	}

	if _, err := parseConfigYAML(configYAML); err != nil {
		return jsonResult(ValidateConfigResult{
			Valid: false,
			Error: err.Error(),
		})
	}

	return jsonResult(ValidateConfigResult{Valid: true})
}
