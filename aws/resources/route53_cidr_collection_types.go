package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// Route53CidrCollection - represents all Route53CidrCollection
type Route53CidrCollection struct {
	BaseAwsResource
	Client route53iface.Route53API
	Region string
	Ids    []string
}

func (r *Route53CidrCollection) Init(session *session.Session) {
	r.Client = route53.New(session)
}

// ResourceName - the simple name of the aws resource
func (r *Route53CidrCollection) ResourceName() string {
	return "route53-cidr-collection"
}

func (r *Route53CidrCollection) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the cidr collections
func (r *Route53CidrCollection) ResourceIdentifiers() []string {
	return r.Ids
}

func (r *Route53CidrCollection) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := r.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	r.Ids = awsgo.StringValueSlice(identifiers)
	return r.Ids, nil
}

// Nuke - nuke 'em all!!!
func (r *Route53CidrCollection) Nuke(identifiers []string) error {
	if err := r.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
