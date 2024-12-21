package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ACMServiceAPI interface {
	DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
	ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
}

// ACM - represents all ACM
type ACM struct {
	BaseAwsResource
	Client ACMServiceAPI
	Region string
	ARNs   []string
}

func (a *ACM) InitV2(cfg aws.Config) {
	a.Client = acm.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (a *ACM) ResourceName() string {
	return "acm"
}

// ResourceIdentifiers - the arns of the aws certificate manager certificates
func (a *ACM) ResourceIdentifiers() []string {
	return a.ARNs
}

func (a *ACM) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

// GetAndSetResourceConfig To get the resource configuration
func (a *ACM) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ACM
}

func (a *ACM) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := a.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	a.ARNs = aws.ToStringSlice(identifiers)
	return a.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (a *ACM) Nuke(arns []string) error {
	if err := a.nukeAll(aws.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
