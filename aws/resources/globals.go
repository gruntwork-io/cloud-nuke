package resources

import "time"

const (
	// A tag used to set custom AWS Tags to resources that do not support `created at` timestamp> - EIP & ECS Clusters.
	// This is used in relation to the `--older-than <duration>` filtering that `cloud-nuke` allows.
	// Due to its destructive nature, `cloud-nuke` has been configured not to delete AWS resources without known creation time,
	// and instead tag them with the `firstSeenTagKey`.
	// The next time `cloud-nuke aws --older-than <duration>` is run, it will use the tag to determine if the AWS resource should be deleted or not.
	firstSeenTagKey = "cloud-nuke-first-seen"

	// The time format of the `firstSeenTagKey` tag value.
	firstSeenTimeFormat = time.RFC3339
)
