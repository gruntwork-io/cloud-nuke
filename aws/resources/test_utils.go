package resources

import (
	"testing"

	terratestaws "github.com/gruntwork-io/terratest/modules/aws"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/require"
)

func GetTestSession(t *testing.T, approvedRegions []string, forbiddenRegions []string) *session.Session {
	region := terratestaws.GetRandomStableRegion(t, approvedRegions, forbiddenRegions)
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	return session
}
