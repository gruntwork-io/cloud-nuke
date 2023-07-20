package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type SNS struct {
	Client snsiface.SNSAPI
	Region string
	Arns   []string
}

func (s SNS) ResourceName() string {
	return "sns"
}

func (s SNS) ResourceIdentifiers() []string {
	return s.Arns
}

func (s SNS) MaxBatchSize() int {
	return 50
}

func (s SNS) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSNSTopics(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// custom errors

type TooManySNSTopicsErr struct{}

func (err TooManySNSTopicsErr) Error() string {
	return "Too many SNS Topics requested at once."
}
