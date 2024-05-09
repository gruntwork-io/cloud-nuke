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

// Route53HostedZone - represents all Route53HostedZone
type Route53HostedZone struct {
	BaseAwsResource
	Client route53iface.Route53API
	Region string
	Ids    []string
}

func (r *Route53HostedZone) Init(session *session.Session) {
	r.Client = route53.New(session)
}

// ResourceName - the simple name of the aws resource
func (r *Route53HostedZone) ResourceName() string {
	return "route53-hosted-zone"
}

func (r *Route53HostedZone) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids hosted zones
func (r *Route53HostedZone) ResourceIdentifiers() []string {
	return r.Ids
}

func (r *Route53HostedZone) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Route53HostedZone
}

func (r *Route53HostedZone) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := r.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	r.Ids = awsgo.StringValueSlice(identifiers)
	return r.Ids, nil
}

// Nuke - nuke 'em all!!!
func (r *Route53HostedZone) Nuke(identifiers []string) error {
	if err := r.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
