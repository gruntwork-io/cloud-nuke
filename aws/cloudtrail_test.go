package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCloudTrailTrails(t *testing.T) {
	t.Parallel()

	session, err := getAwsSession(false)

	require.NoError(t, err)

	trailArn := createCloudTrailTrail(t, session)
	defer deleteCloudTrailTrail(t, session, trailArn, false)

	trailArns, err := getAllCloudtrailTrails(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(trailArns), aws.StringValue(trailArn))
}

func deleteCloudTrailTrail(t *testing.T, session *session.Session, trailARN *string, checkErr bool) {
	cloudtrailSvc := cloudtrail.New(session)

	param := &cloudtrail.DeleteTrailInput{
		Name: trailARN,
	}

	_, deleteErr := cloudtrailSvc.DeleteTrail(param)
	if checkErr {
		require.NoError(t, deleteErr)
	}
}

func createCloudTrailTrail(t *testing.T, session *session.Session) *string {
	cloudtrailSvc := cloudtrail.New(session)
	s3Svc := s3.New(session)
	stsSvc := sts.New(session)

	name := strings.ToLower(fmt.Sprintf("cloud-nuke-test-%s-%s", util.UniqueID(), util.UniqueID()))

	logging.Logger.Debugf("Bucket: %s - creating", name)

	_, bucketCreateErr := s3Svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(name),
	})

	require.NoError(t, bucketCreateErr)

	waitErr := s3Svc.WaitUntilBucketExists(
		&s3.HeadBucketInput{
			Bucket: aws.String(name),
		},
	)

	require.NoError(t, waitErr)

	// Create and attach the expected S3 bucket policy that CloudTrail requires
	policyJson := `
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AWSCloudTrailAclCheck20150319",
            "Effect": "Allow",
            "Principal": {"Service": "cloudtrail.amazonaws.com"},
            "Action": "s3:GetBucketAcl",
            "Resource": "arn:aws:s3:::%s",
            "Condition": {
                "StringEquals": {
                    "aws:SourceArn": "arn:aws:cloudtrail:%s:%s:trail/%s"
                }
            }
        },
        {
            "Sid": "AWSCloudTrailWrite20150319",
            "Effect": "Allow",
            "Principal": {"Service": "cloudtrail.amazonaws.com"},
            "Action": "s3:PutObject",
            "Resource": "arn:aws:s3:::%s/AWSLogs/%s/*",
            "Condition": {
                "StringEquals": {
                    "s3:x-amz-acl": "bucket-owner-full-control",
                    "aws:SourceArn": "arn:aws:cloudtrail:%s:%s:trail/%s"
                }
            }
        }
    ]
}
`

	// Look up the current account ID so that we can interpolate it in the S3 bucket policy
	callerIdInput := &sts.GetCallerIdentityInput{}

	result, err := stsSvc.GetCallerIdentity(callerIdInput)

	require.NoError(t, err)

	renderedJson := fmt.Sprintf(
		policyJson,
		name,
		*session.Config.Region,
		aws.StringValue(result.Account),
		name,
		name,
		aws.StringValue(result.Account),
		*session.Config.Region,
		aws.StringValue(result.Account),
		name,
	)

	_, err = s3Svc.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: aws.String(name),
		Policy: aws.String(strings.TrimSpace(renderedJson)),
	})

	require.NoError(t, err)

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)

	param := &cloudtrail.CreateTrailInput{
		Name:         aws.String(name),
		S3BucketName: aws.String(name),
	}

	output, createTrailErr := cloudtrailSvc.CreateTrail(param)
	require.NoError(t, createTrailErr)

	return output.TrailARN
}

func TestNukeCloudTrailOne(t *testing.T) {
	t.Parallel()
	session, err := getAwsSession(false)

	require.NoError(t, err)

	trailArn := createCloudTrailTrail(t, session)
	defer deleteCloudTrailTrail(t, session, trailArn, false)

	identifiers := []*string{trailArn}

	require.NoError(
		t,
		nukeAllCloudTrailTrails(session, identifiers),
	)

	assertCloudTrailTrailsDeleted(t, session, identifiers)
}

func TestNukeCloudTrailTrailMoreThanOne(t *testing.T) {
	t.Parallel()

	session, err := getAwsSession(false)

	require.NoError(t, err)

	trailArns := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		trailArn := createCloudTrailTrail(t, session)
		defer deleteCloudTrailTrail(t, session, trailArn, false)
		trailArns = append(trailArns, trailArn)
	}

	require.NoError(
		t,
		nukeAllCloudTrailTrails(session, trailArns),
	)

	assertCloudTrailTrailsDeleted(t, session, trailArns)
}

func assertCloudTrailTrailsDeleted(t *testing.T, session *session.Session, identifiers []*string) {
	svc := cloudtrail.New(session)

	resp, err := svc.DescribeTrails(&cloudtrail.DescribeTrailsInput{
		TrailNameList: identifiers,
	})
	require.NoError(t, err)
	if len(resp.TrailList) > 0 {
		t.Fatalf("At least one of the following CloudTrail Trails was not deleted: %+v\n", aws.StringValueSlice(identifiers))
	}
}
