package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/collections"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	terraws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/go-commons/retry"
)

func TestListSecretsManagerSecrets(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	region = "ap-northeast-3"
	require.NoError(t, err)

	secretName := fmt.Sprintf("test-cloud-nuke-secretsmanager-list-%s", random.UniqueId())
	defer terraws.DeleteSecret(t, region, secretName, true)
	arn := createSecretStringWithDefaultKey(t, region, secretName)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	secretARNPtrs, err := getAllSecretsManagerSecrets(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(secretARNPtrs), arn)
}

func TestTimeFilterExclusionNewlyCreatedSecret(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	secretName := fmt.Sprintf("test-cloud-nuke-secretsmanager-exclusion-%s", random.UniqueId())
	defer terraws.DeleteSecret(t, region, secretName, true)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	// Assert user didn't exist
	secretARNPtrs, err := getAllSecretsManagerSecrets(session, time.Now(), config.Config{})
	require.NoError(t, err)
	for _, secretARNPtr := range secretARNPtrs {
		assert.NotContains(t, aws.StringValue(secretARNPtr), secretName)
	}

	// Creates a secret
	arn := createSecretStringWithDefaultKey(t, region, secretName)

	// Assert secret is picked up without filters
	secretARNPtrsNewer, err := getAllSecretsManagerSecrets(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(secretARNPtrsNewer), arn)

	// Assert user doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	secretARNPtrsOlder, err := getAllSecretsManagerSecrets(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(secretARNPtrsOlder), arn)
}

func TestNukeSecretOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	secretName := fmt.Sprintf("test-cloud-nuke-secretsmanager-one-%s", random.UniqueId())
	// We use the E version and ignore the error, as this is meant to be a stop gap deletion in case nuke has a bug.
	defer terraws.DeleteSecretE(t, region, secretName, true)
	arn := createSecretStringWithDefaultKey(t, region, secretName)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	require.NoError(
		t,
		nukeAllSecretsManagerSecrets(session, aws.StringSlice([]string{arn})),
	)

	// Make sure the secret is deleted.
	_, err = terraws.GetSecretValueE(t, region, arn)
	assert.Error(t, err)
}

func TestNukeSecretMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	secretNameBase := fmt.Sprintf("test-cloud-nuke-secretsmanager-more-than-one-%s", random.UniqueId())

	secretArns := []string{}
	for i := 0; i < 3; i++ {
		secretName := fmt.Sprintf("%s-%d", secretNameBase, i)
		// We use the E version and ignore the error, as this is meant to be a stop gap deletion in case nuke has a bug.
		defer terraws.DeleteSecretE(t, region, secretName, true)
		secretArns = append(secretArns, createSecretStringWithDefaultKey(t, region, secretName))
	}

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	require.NoError(
		t,
		nukeAllSecretsManagerSecrets(session, aws.StringSlice(secretArns)),
	)

	// Make sure the secret is deleted.
	for _, arn := range secretArns {
		_, err = terraws.GetSecretValueE(t, region, arn)
		assert.Error(t, err)
	}
}

// Helper functions for driving the secrets manager tests

// createSecretStringWithDefaultKey creates a new secret with a random value in Secrets Manager using the default
// "aws/secretsmanager" KMS key and returns the secret ARN
func createSecretStringWithDefaultKey(t *testing.T, awsRegion string, name string) string {
	description := "Random secret created for cloud-nuke testing."
	secretVal := random.UniqueId()
	arn := terraws.CreateSecretStringWithDefaultKey(t, awsRegion, description, name, secretVal)
	// Check if created secret is available by checking secret ARN in list of all secrets
	// https://github.com/gruntwork-io/cloud-nuke/issues/227
	awsSession, err := session.NewSession(&aws.Config{Region: aws.String(awsRegion)})
	require.NoError(t, err)
	err = retry.DoWithRetry(
		logging.Logger,
		"Check if created secret is available",
		4,
		15*time.Second,
		func() error {
			secretARNPtrs, err := getAllSecretsManagerSecrets(awsSession, time.Now(), config.Config{})
			if err != nil {
				return err
			}
			if collections.ListContainsElement(aws.StringValueSlice(secretARNPtrs), arn) {
				return nil
			}
			return fmt.Errorf("not found secret %s", arn)
		},
	)
	require.NoError(t, err)
	return arn
}
