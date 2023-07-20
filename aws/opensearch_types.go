package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// OpenSearchDomain represents all OpenSearch domains found in a region
type OpenSearchDomain struct {
	Client      iamiface.IAMAPI
	Region      string
	DomainNames []string
}

// ResourceName is the simple name of the aws resource
func (domains OpenSearchDomain) ResourceName() string {
	return "open-search-domain"
}

// ResourceIdentifiers the collected OpenSearch Domains
func (domains OpenSearchDomain) ResourceIdentifiers() []string {
	return domains.DomainNames
}

// MaxBatchSize returns the number of resources that should be nuked at a time. A small number is used to ensure AWS
// doesn't throttle. OpenSearch Domains do not support bulk delete, so we will be deleting this many in parallel
// using go routines. We conservatively pick 10 here, both to limit overloading the runtime and to avoid AWS throttling
// with many API calls.
func (domains OpenSearchDomain) MaxBatchSize() int {
	return 10
}

// Nuke nukes all OpenSearch domain resources
func (domains OpenSearchDomain) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllOpenSearchDomains(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
