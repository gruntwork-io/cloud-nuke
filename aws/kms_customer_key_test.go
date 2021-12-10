package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListKmsUserKeys(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	createdKeyId := createKmsCustomerManagedKey(t, session, err)

	// test if listing of keys will return new key
	keys, err := getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now())
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(keys), createdKeyId)

	// test if time shift works
	olderThan := time.Now().Add(-1 * time.Hour)
	keys, err = getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), olderThan)
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(keys), createdKeyId)
}

func TestRemoveKmsUserKeys(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	createdKeyId := createKmsCustomerManagedKey(t, session, err)

	err = nukeAllCustomerManagedKmsKeys(session, []*string{&createdKeyId})
	require.NoError(t, err)

	// test if key is not included for removal second time
	keys, err := getAllKmsUserKeys(session, KmsCustomerKeys{}.MaxBatchSize(), time.Now())
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(keys), createdKeyId)
}

func createKmsCustomerManagedKey(t *testing.T, session *session.Session, err error) string {
	svc := kms.New(session)
	input := &kms.CreateKeyInput{}
	result, err := svc.CreateKey(input)
	require.NoError(t, err)
	createdKeyId := *result.KeyMetadata.KeyId
	time.Sleep(15 * time.Second)
	return createdKeyId
}
