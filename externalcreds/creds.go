package externalcreds

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// configProvider is an optional override for AWS config creation.
// When non-nil, Get delegates to this function instead of using LoadDefaultConfig.
var configProvider func(region string) (aws.Config, error)

// SetConfigProvider overrides the default AWS config creation used by cloud-nuke.
// This is useful when importing cloud-nuke as a library and retrieving AWS
// credentials at runtime (e.g., assume-role, vault, custom credential providers).
//
// Pass nil to restore the default behavior (config.LoadDefaultConfig).
// This should be called once at startup, before any cloud-nuke operations.
func SetConfigProvider(fn func(region string) (aws.Config, error)) {
	configProvider = fn
}

func Get(region string) (aws.Config, error) {
	if configProvider != nil {
		return configProvider(region)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)

	if err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}
