package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedSecretsManager struct {
	secretsmanageriface.SecretsManagerAPI
	ListSecretsOutput                  secretsmanager.ListSecretsOutput
	DescribeSecretOutput               secretsmanager.DescribeSecretOutput
	DeleteSecretOutput                 secretsmanager.DeleteSecretOutput
	RemoveRegionsFromReplicationOutput secretsmanager.RemoveRegionsFromReplicationOutput
}

func (m mockedSecretsManager) ListSecretsPages(input *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	fn(&m.ListSecretsOutput, true)
	return nil
}

func (m mockedSecretsManager) DescribeSecret(input *secretsmanager.DescribeSecretInput) (*secretsmanager.DescribeSecretOutput, error) {
	return &m.DescribeSecretOutput, nil
}

func (m mockedSecretsManager) DeleteSecret(input *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	return &m.DeleteSecretOutput, nil
}

func (m mockedSecretsManager) RemoveRegionsFromReplication(input *secretsmanager.RemoveRegionsFromReplicationInput) (*secretsmanager.RemoveRegionsFromReplicationOutput, error) {
	return &m.RemoveRegionsFromReplicationOutput, nil
}

func TestSecretsManagerSecrets_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-name-1"
	testName2 := "test-name-2"
	testArn1 := "test-arn1"
	testArn2 := "test-arn2"
	now := time.Now()
	sms := SecretsManagerSecrets{
		Client: mockedSecretsManager{
			ListSecretsOutput: secretsmanager.ListSecretsOutput{
				SecretList: []*secretsmanager.SecretListEntry{
					{
						Name:        aws.String(testName1),
						ARN:         aws.String(testArn1),
						CreatedDate: &now,
					},
					{
						Name:        aws.String(testName2),
						ARN:         aws.String(testArn2),
						CreatedDate: aws.Time(now.Add(1)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := sms.getAll(context.Background(), config.Config{
				SecretsManagerSecrets: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestSecretsManagerSecrets_NukeAll(t *testing.T) {

	t.Parallel()

	sms := SecretsManagerSecrets{
		Client: mockedSecretsManager{
			DescribeSecretOutput: secretsmanager.DescribeSecretOutput{
				ARN: aws.String("test-arn"),
			},
			DeleteSecretOutput: secretsmanager.DeleteSecretOutput{},
			RemoveRegionsFromReplicationOutput: secretsmanager.RemoveRegionsFromReplicationOutput{
				ARN: aws.String("test-arn"),
			},
		},
	}

	err := sms.nukeAll([]*string{aws.String("test-arn")})
	require.NoError(t, err)
}
