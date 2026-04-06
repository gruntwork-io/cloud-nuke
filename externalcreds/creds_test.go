package externalcreds

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_DefaultBehavior(t *testing.T) {
	// Ensure no custom provider is set
	SetConfigProvider(nil)

	cfg, err := Get("us-west-2")
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", cfg.Region)
}

func TestGet_CustomProvider(t *testing.T) {
	called := false
	SetConfigProvider(func(region string) (aws.Config, error) {
		called = true
		return aws.Config{Region: region}, nil
	})
	t.Cleanup(func() { SetConfigProvider(nil) })

	cfg, err := Get("eu-west-1")
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "eu-west-1", cfg.Region)
}

func TestGet_CustomProviderError(t *testing.T) {
	SetConfigProvider(func(region string) (aws.Config, error) {
		return aws.Config{}, fmt.Errorf("custom error")
	})
	t.Cleanup(func() { SetConfigProvider(nil) })

	_, err := Get("us-east-1")
	assert.EqualError(t, err, "custom error")
}

func TestGet_NilProviderRestoresDefault(t *testing.T) {
	SetConfigProvider(func(region string) (aws.Config, error) {
		return aws.Config{Region: "custom"}, nil
	})

	// Reset to default
	SetConfigProvider(nil)

	cfg, err := Get("ap-southeast-1")
	require.NoError(t, err)
	assert.Equal(t, "ap-southeast-1", cfg.Region)
}
