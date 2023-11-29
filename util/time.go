package util

import (
	"time"

	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	// FirstSeenTagKey A tag used to set custom AWS Tags to resources that do not support `created at` timestamp> - EIP & ECS Clusters.
	// This is used in relation to the `--older-than <duration>` filtering that `cloud-nuke` allows.
	// Due to its destructive nature, `cloud-nuke` has been configured not to delete AWS resources without known creation time,
	// and instead tag them with the `firstSeenTagKey`.
	// The next time `cloud-nuke aws --older-than <duration>` is run, it will use the tag to determine if the AWS resource should be deleted or not.
	FirstSeenTagKey = "cloud-nuke-first-seen"

	// The time format of the `firstSeenTagKey` tag value.
	firstSeenTimeFormat = time.RFC3339
)

func IsFirstSeenTag(key *string) bool {
	return aws.StringValue(key) == FirstSeenTagKey
}

func ParseTimestamp(timestamp *string) (*time.Time, error) {
	parsed, err := time.Parse(firstSeenTimeFormat, aws.StringValue(timestamp))
	if err != nil {
		logging.Debugf("Error parsing the timestamp into a `RFC3339` Time format")
		return nil, errors.WithStackTrace(err)

	}

	return &parsed, nil
}

func FormatTimestamp(timestamp time.Time) string {
	return timestamp.Format(firstSeenTimeFormat)
}
