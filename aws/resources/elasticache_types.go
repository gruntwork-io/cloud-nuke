package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// Elasticaches - represents all Elasticache clusters
type Elasticaches struct {
	Client     elasticacheiface.ElastiCacheAPI
	Region     string
	ClusterIds []string
}

func (cache *Elasticaches) Init(session *session.Session) {
	cache.Client = elasticache.New(session)
}

// ResourceName - the simple name of the aws resource
func (cache *Elasticaches) ResourceName() string {
	return "elasticache"
}

// ResourceIdentifiers - The instance ids of the elasticache clusters
func (cache *Elasticaches) ResourceIdentifiers() []string {
	return cache.ClusterIds
}

func (cache *Elasticaches) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (cache *Elasticaches) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cache.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cache.ClusterIds = awsgo.StringValueSlice(identifiers)
	return cache.ClusterIds, nil
}

// Nuke - nuke 'em all!!!
func (cache *Elasticaches) Nuke(identifiers []string) error {
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

func (pg *ElasticacheParameterGroups) Init(session *session.Session) {
	pg.Client = elasticache.New(session)
}

// ResourceName - the simple name of the aws resource
func (pg *ElasticacheParameterGroups) ResourceName() string {
	return "elasticacheParameterGroups"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (pg *ElasticacheParameterGroups) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg *ElasticacheParameterGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (pg *ElasticacheParameterGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pg.GroupNames = awsgo.StringValueSlice(identifiers)
	return pg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (pg *ElasticacheParameterGroups) Nuke(identifiers []string) error {
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

func (sg *ElasticacheSubnetGroups) Init(session *session.Session) {
	sg.Client = elasticache.New(session)
}

func (sg *ElasticacheSubnetGroups) ResourceName() string {
	return "elasticacheSubnetGroups"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (sg *ElasticacheSubnetGroups) ResourceIdentifiers() []string {
	return sg.GroupNames
}

func (sg *ElasticacheSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (sg *ElasticacheSubnetGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sg.GroupNames = awsgo.StringValueSlice(identifiers)
	return sg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (sg *ElasticacheSubnetGroups) Nuke(identifiers []string) error {
	if err := sg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
