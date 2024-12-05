package util

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"
	"github.com/gruntwork-io/cloud-nuke/logging"
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
	firstSeenTimeFormat       = time.RFC3339
	firstSeenTimeFormatLegacy = time.DateTime
)

func IsFirstSeenTag(key *string) bool {
	return aws.StringValue(key) == FirstSeenTagKey
}

func ParseTimestamp(timestamp *string) (*time.Time, error) {
	parsed, err := time.Parse(firstSeenTimeFormat, aws.StringValue(timestamp))
	if err != nil {
		logging.Debugf("Error parsing the timestamp into a `RFC3339` Time format. Trying parsing the timestamp using the legacy `time.DateTime` format.")
		parsed, err = time.Parse(firstSeenTimeFormatLegacy, aws.StringValue(timestamp))
		if err != nil {
			logging.Debugf("Error parsing the timestamp into legacy `time.DateTime` Time format")
			return nil, errors.WithStackTrace(err)
		}
	}

	return &parsed, nil
}

func FormatTimestamp(timestamp time.Time) string {
	return timestamp.Format(firstSeenTimeFormat)
}

func GetOrCreateFirstSeen(ctx context.Context, client interface{}, identifier *string, tags map[string]string) (*time.Time, error) {

	var firstSeenTime *time.Time
	var err error

	excludeFirstSeenTag, err := GetBoolFromContext(ctx, ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if excludeFirstSeenTag {
		return nil, nil
	}

	// check the first seen already exists in the given map
	for key, value := range tags {
		if IsFirstSeenTag(awsgo.String(key)) {
			firstSeenTime, err = ParseTimestamp(awsgo.String(value))
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
		}
	}

	if firstSeenTime == nil {
		now := time.Now().UTC()
		firstSeenTime = &now

		switch v := client.(type) {
		case ec2iface.EC2API:
			_, err = v.CreateTags(&ec2.CreateTagsInput{
				Resources: []*string{identifier},
				Tags: []*ec2.Tag{
					{
						Key:   awsgo.String(FirstSeenTagKey),
						Value: awsgo.String(FormatTimestamp(now)),
					},
				},
			})
		case networkfirewalliface.NetworkFirewallAPI:
			_, err = v.TagResource(&networkfirewall.TagResourceInput{
				ResourceArn: identifier,
				Tags: []*networkfirewall.Tag{
					{
						Key:   awsgo.String(FirstSeenTagKey),
						Value: awsgo.String(FormatTimestamp(now)),
					},
				},
			})
		default:
			return nil, errors.WithStackTrace(fmt.Errorf("invalid type %v for first seen tag", v))
		}
	}

	return firstSeenTime, err
}
