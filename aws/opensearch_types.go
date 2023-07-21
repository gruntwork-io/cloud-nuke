package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// OpenSearchDomains represents all OpenSearch domains found in a region
type OpenSearchDomains struct {
	Client      iamiface.IAMAPI
	Region      string
	DomainNames []string
}

// ResourceName is the simple name of the aws resource
func (domains OpenSearchDomains) ResourceName() string {
	return "opensearchdomain"
}

// ResourceIdentifiers the collected OpenSearch Domains
func (domains OpenSearchDomains) ResourceIdentifiers() []string {
	return domains.DomainNames
}

// MaxBatchSize returns the number of resources that should be nuked at a time. A small number is used to ensure AWS
// doesn't throttle. OpenSearch Domains do not support bulk delete, so we will be deleting this many in parallel
// using go routines. We conservatively pick 10 here, both to limit overloading the runtime and to avoid AWS throttling
// with many API calls.
func (domains OpenSearchDomains) MaxBatchSize() int {
	return 10
}

// Nuke nukes all OpenSearch domain resources
func (domains OpenSearchDomains) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllOpenSearchDomains(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
