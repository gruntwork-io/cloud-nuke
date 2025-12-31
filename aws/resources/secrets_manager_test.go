package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockSecretsManagerClient struct {
	ListSecretsOutput                  secretsmanager.ListSecretsOutput
	DescribeSecretOutput               secretsmanager.DescribeSecretOutput
	DeleteSecretOutput                 secretsmanager.DeleteSecretOutput
	RemoveRegionsFromReplicationOutput secretsmanager.RemoveRegionsFromReplicationOutput
}

func (m *mockSecretsManagerClient) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	return &m.ListSecretsOutput, nil
}

func (m *mockSecretsManagerClient) DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
	return &m.DescribeSecretOutput, nil
}

func (m *mockSecretsManagerClient) DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	return &m.DeleteSecretOutput, nil
}

func (m *mockSecretsManagerClient) RemoveRegionsFromReplication(ctx context.Context, params *secretsmanager.RemoveRegionsFromReplicationInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.RemoveRegionsFromReplicationOutput, error) {
	return &m.RemoveRegionsFromReplicationOutput, nil
}

func TestListSecretsManagerSecrets(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockSecretsManagerClient{
		ListSecretsOutput: secretsmanager.ListSecretsOutput{
			SecretList: []types.SecretListEntry{
				{Name: aws.String("secret1"), ARN: aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:secret1"), CreatedDate: aws.Time(now)},
				{Name: aws.String("secret2"), ARN: aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:secret2"), CreatedDate: aws.Time(now)},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected: []string{
				"arn:aws:secretsmanager:us-east-1:123456789012:secret:secret1",
				"arn:aws:secretsmanager:us-east-1:123456789012:secret:secret2",
			},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("secret1")}},
				},
			},
			expected: []string{"arn:aws:secretsmanager:us-east-1:123456789012:secret:secret2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listSecretsManagerSecrets(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSecretsManagerSecret(t *testing.T) {
	t.Parallel()

	mock := &mockSecretsManagerClient{
		DescribeSecretOutput: secretsmanager.DescribeSecretOutput{
			ARN: aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret"),
		},
	}

	err := deleteSecretsManagerSecret(context.Background(), mock, aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret"))
	require.NoError(t, err)
}
