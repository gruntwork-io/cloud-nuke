package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SNSTopic struct {
	Client snsiface.SNSAPI
	Region string
	Arns   []string
}

func (s *SNSTopic) Init(session *session.Session) {
	s.Client = sns.New(session)
}

func (s *SNSTopic) ResourceName() string {
	return "snstopic"
}

func (s *SNSTopic) ResourceIdentifiers() []string {
	return s.Arns
}

func (s *SNSTopic) MaxBatchSize() int {
	return 50
}

func (s *SNSTopic) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := s.getAll(configObj)
	if err != nil {
		return nil, err
	}

	s.Arns = awsgo.StringValueSlice(identifiers)
	return s.Arns, nil
}

func (s *SNSTopic) Nuke(identifiers []string) error {
	if err := s.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// custom errors

type TooManySNSTopicsErr struct{}

func (err TooManySNSTopicsErr) Error() string {
	return "Too many SNS Topics requested at once."
}
