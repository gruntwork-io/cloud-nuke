package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// Elasticache - represents all Elasticache clusters
type Elasticache struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	ClusterIds []string
}

// ResourceName - the simple name of the aws resource
func (cache Elasticache) ResourceName() string {
	return "elasticache"
}

// ResourceIdentifiers - The instance ids of the elasticache clusters
func (cache Elasticache) ResourceIdentifiers() []string {
	return cache.ClusterIds
}

func (cache Elasticache) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (cache Elasticache) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticacheClusters(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

/*
Elasticache Parameter Groups
*/

type ElasticacheParameterGroup struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	GroupNames []string
}

// ResourceName - the simple name of the aws resource
func (pg ElasticacheParameterGroup) ResourceName() string {
	return "elasticache-parameter-group"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (pg ElasticacheParameterGroup) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg ElasticacheParameterGroup) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (pg ElasticacheParameterGroup) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticacheParameterGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

/*
Elasticache Subnet Groups
*/

type ElasticacheSubnetGroup struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	GroupNames []string
}

func (sg ElasticacheSubnetGroup) ResourceName() string {
	return "elasticache-subnet-group"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (sg ElasticacheSubnetGroup) ResourceIdentifiers() []string {
	return sg.GroupNames
}

func (sg ElasticacheSubnetGroup) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (sg ElasticacheSubnetGroup) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticacheSubnetGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
