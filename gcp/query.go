package gcp

import "time"

// Query represents the desired parameters for scanning GCP resources.
// This mirrors the aws.Query struct for interface consistency.
type Query struct {
	ProjectID            string
	ResourceTypes        []string
	ExcludeResourceTypes []string
	ExcludeAfter         *time.Time
	IncludeAfter         *time.Time
	Timeout              *time.Duration
}
