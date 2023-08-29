package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/aws/aws-sdk-go/service/kafka/kafkaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// MSKCluster - represents all AWS Managed Streaming for Kafka clusters that should be deleted.
type MSKCluster struct {
	Client      kafkaiface.KafkaAPI
	Region      string
	ClusterArns []string
}

func (msk MSKCluster) Init(session *session.Session) {
	msk.Client = kafka.New(session)
}

// ResourceName - the simple name of the aws resource
func (msk MSKCluster) ResourceName() string {
	return "msk-cluster"
}

// ResourceIdentifiers - The instance ids of the AWS Managed Streaming for Kafka clusters
func (msk MSKCluster) ResourceIdentifiers() []string {
	return msk.ClusterArns
}

func (msk MSKCluster) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (msk MSKCluster) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := msk.getAll(configObj)
	if err != nil {
		return nil, err
	}

	msk.ClusterArns = awsgo.StringValueSlice(identifiers)
	return msk.ClusterArns, nil
}

// Nuke - nuke 'em all!!!
func (msk MSKCluster) Nuke(_ *session.Session, identifiers []string) error {
	if err := msk.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
