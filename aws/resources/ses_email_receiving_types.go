package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SesReceiptFilter struct {
	BaseAwsResource
	Client  sesiface.SESAPI
	Region  string
	Ids     []string
	Nukable map[string]bool
}

func (sef *SesReceiptFilter) Init(session *session.Session) {
	sef.Client = ses.New(session)
	sef.Nukable = map[string]bool{}
}

// ResourceName - the simple name of the aws resource
func (sef *SesReceiptFilter) ResourceName() string {
	return "ses-receipt-filter"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (sef *SesReceiptFilter) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the receipt filter
func (sef *SesReceiptFilter) ResourceIdentifiers() []string {
	return sef.Ids
}

func (sef *SesReceiptFilter) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESReceiptFilter
}

func (sef *SesReceiptFilter) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sef.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sef.Ids = awsgo.StringValueSlice(identifiers)
	return sef.Ids, nil
}

// Nuke - nuke 'em all!!!
func (sef *SesReceiptFilter) Nuke(identifiers []string) error {
	if err := sef.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// SesReceiptRule - represents all ses receipt rules
type SesReceiptRule struct {
	BaseAwsResource
	Client  sesiface.SESAPI
	Region  string
	Ids     []string
	Nukable map[string]bool
}

func (ser *SesReceiptRule) Init(session *session.Session) {
	ser.Client = ses.New(session)
	ser.Nukable = map[string]bool{}
}

// ResourceName - the simple name of the aws resource
func (ser *SesReceiptRule) ResourceName() string {
	return "ses-receipt-rule-set"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (ser *SesReceiptRule) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The names of the rule set
func (ser *SesReceiptRule) ResourceIdentifiers() []string {
	return ser.Ids
}

func (sef *SesReceiptRule) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESReceiptRuleSet
}

func (ser *SesReceiptRule) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ser.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ser.Ids = awsgo.StringValueSlice(identifiers)
	return ser.Ids, nil
}

// Nuke - nuke 'em all!!!
func (ser *SesReceiptRule) Nuke(identifiers []string) error {
	if err := ser.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
