package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// Elasticaches - represents all Elasticache clusters
type Elasticaches struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	ClusterIds []string
}

// ResourceName - the simple name of the aws resource
func (cache Elasticaches) ResourceName() string {
	return "elasticache"
}

// ResourceIdentifiers - The instance ids of the elasticache clusters
func (cache Elasticaches) ResourceIdentifiers() []string {
	return cache.ClusterIds
}

func (cache Elasticaches) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (cache Elasticaches) Nuke(session *session.Session, identifiers []string) error {
	if err := cache.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

/*
Elasticache Parameter Groups
*/

type ElasticacheParameterGroups struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	GroupNames []string
}

// ResourceName - the simple name of the aws resource
func (pg ElasticacheParameterGroups) ResourceName() string {
	return "elasticacheParameterGroups"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (pg ElasticacheParameterGroups) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg ElasticacheParameterGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (pg ElasticacheParameterGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := pg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

/*
Elasticache Subnet Groups
*/

type ElasticacheSubnetGroups struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	GroupNames []string
}

func (sg ElasticacheSubnetGroups) ResourceName() string {
	return "elasticacheSubnetGroups"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (sg ElasticacheSubnetGroups) ResourceIdentifiers() []string {
	return sg.GroupNames
}

func (sg ElasticacheSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (sg ElasticacheSubnetGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := sg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
