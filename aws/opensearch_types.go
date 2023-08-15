package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/opensearchservice"
	"github.com/aws/aws-sdk-go/service/opensearchservice/opensearchserviceiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// OpenSearchDomains represents all OpenSearch domains found in a region
type OpenSearchDomains struct {
	Client      opensearchserviceiface.OpenSearchServiceAPI
	Region      string
	DomainNames []string
}

func (osd *OpenSearchDomains) Init(session *session.Session) {
	osd.Client = opensearchservice.New(session)
}

// ResourceName is the simple name of the aws resource
func (osd *OpenSearchDomains) ResourceName() string {
	return "opensearchdomain"
}

// ResourceIdentifiers the collected OpenSearch Domains
func (osd *OpenSearchDomains) ResourceIdentifiers() []string {
	return osd.DomainNames
}

// MaxBatchSize returns the number of resources that should be nuked at a time. A small number is used to ensure AWS
// doesn't throttle. OpenSearch Domains do not support bulk delete, so we will be deleting this many in parallel
// using go routines. We conservatively pick 10 here, both to limit overloading the runtime and to avoid AWS throttling
// with many API calls.
func (osd *OpenSearchDomains) MaxBatchSize() int {
	return 10
}

func (osd *OpenSearchDomains) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := osd.getAll(configObj)
	if err != nil {
		return nil, err
	}

	osd.DomainNames = awsgo.StringValueSlice(identifiers)
	return osd.DomainNames, nil
}

// Nuke nukes all OpenSearch domain resources
func (osd *OpenSearchDomains) Nuke(identifiers []string) error {
	if err := osd.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
