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

// Route53TrafficPolicy - represents all Route53TrafficPolicy
type Route53TrafficPolicy struct {
	BaseAwsResource
	Client     route53iface.Route53API
	Region     string
	Ids        []string
	versionMap map[string]*int64
}

func (r *Route53TrafficPolicy) Init(session *session.Session) {
	r.Client = route53.New(session)
	r.versionMap = make(map[string]*int64)
}

// ResourceName - the simple name of the aws resource
func (r *Route53TrafficPolicy) ResourceName() string {
	return "route53-traffic-policy"
}

func (r *Route53TrafficPolicy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of traffic policies
func (r *Route53TrafficPolicy) ResourceIdentifiers() []string {
	return r.Ids
}
func (r *Route53TrafficPolicy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Route53TrafficPolicy
}

func (r *Route53TrafficPolicy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := r.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	r.Ids = awsgo.StringValueSlice(identifiers)
	return r.Ids, nil
}

// Nuke - nuke 'em all!!!
func (r *Route53TrafficPolicy) Nuke(identifiers []string) error {
	if err := r.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
