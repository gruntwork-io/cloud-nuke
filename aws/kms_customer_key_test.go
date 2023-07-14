package aws

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListKmsUserKeys(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	aliasName := "cloud-nuke-test-" + util.UniqueID()
	keyAlias := fmt.Sprintf("alias/%s", aliasName)
	createdKeyId := createKmsCustomerManagedKey(t, session)
	_ = createKmsCustomerManagedKeyAlias(t, session, createdKeyId, keyAlias)

	// test if listing of keys will return new key and alias
	keys, aliases, err := getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now(), config.Config{}, false)
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(keys), createdKeyId)
	assert.Contains(t, aliases[createdKeyId], keyAlias)

	// test if time shift works
	olderThan := time.Now().Add(-1 * time.Hour)
	keys, aliases, err = getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), olderThan, config.Config{}, false)
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(keys), createdKeyId)
	assert.NotContains(t, aliases[createdKeyId], keyAlias)

	// test if matching by regexp works
	keys, aliases, err = getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now(), config.Config{
		KMSCustomerKeys: config.KMSCustomerKeyResourceType{
			ResourceType: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{
						{RE: *regexp.MustCompile(fmt.Sprintf("^%s", keyAlias))},
					},
				},
			},
		},
	}, false)
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(keys), createdKeyId)
	assert.Contains(t, aliases[createdKeyId], keyAlias)
	assert.Equal(t, 1, len(keys))

	// test if exclusion by regexp works
	keys, aliases, err = getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now(), config.Config{
		KMSCustomerKeys: config.KMSCustomerKeyResourceType{
			ResourceType: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{
						{RE: *regexp.MustCompile(fmt.Sprintf("^%s", keyAlias))},
					},
				},
			},
		},
	}, false)
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(keys), createdKeyId)
	assert.NotContains(t, aliases[createdKeyId], keyAlias)
}

func TestRemoveKmsUserKeys(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	keyAlias := "alias/cloud-nuke-test-" + util.UniqueID()
	createdKeyId := createKmsCustomerManagedKey(t, session)
	_ = createKmsCustomerManagedKeyAlias(t, session, createdKeyId, keyAlias)

	err = nukeAllCustomerManagedKmsKeys(session, []*string{&createdKeyId}, map[string][]string{"keyid": {keyAlias}})
	require.NoError(t, err)

	// test if key is not included for removal second time, after being marked for deletion
	keys, aliases, err := getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now(), config.Config{}, false)

	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(keys), createdKeyId)
	assert.NotContains(t, aliases[createdKeyId], keyAlias)

	// test that all aliases were deleted from the key, even if the key was successfully marked for deletion
	listedAliases, err := listAliasesForKey(session, &createdKeyId)

	require.NoError(t, err)
	assert.Empty(t, listedAliases)
}

func TestRemoveKmsUserKeysAllowUnaliased(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	createdKeyId := createKmsCustomerManagedKey(t, session)

	err = nukeAllCustomerManagedKmsKeys(session, []*string{&createdKeyId}, map[string][]string{})
	require.NoError(t, err)

	// test if key is not included for removal second time, after being marked for deletion
	keys, aliases, err := getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now(), config.Config{}, true)

	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(keys), createdKeyId)
	assert.NotContains(t, aliases, createdKeyId)
}

func createKmsCustomerManagedKey(t *testing.T, session *session.Session) string {
	svc := kms.New(session)
	input := &kms.CreateKeyInput{}
	result, err := svc.CreateKey(input)
	require.NoError(t, err)
	createdKeyId := *result.KeyMetadata.KeyId

	return createdKeyId
}

func createKmsCustomerManagedKeyAlias(t *testing.T, session *session.Session, keyId string, alias string) string {
	svc := kms.New(session)

	aliasInput := &kms.CreateAliasInput{AliasName: &alias, TargetKeyId: &keyId}
	_, err := svc.CreateAlias(aliasInput)
	require.NoError(t, err)

	return *aliasInput.AliasName
}

func listAliasesForKey(session *session.Session, keyId *string) ([]string, error) {
	svc := kms.New(session)
	input := &kms.ListAliasesInput{KeyId: keyId}
	result, err := svc.ListAliases(input)

	aliases := make([]string, 0)

	for _, alias := range result.Aliases {
		aliases = append(aliases, *alias.TargetKeyId)
	}

	return aliases, err
}
