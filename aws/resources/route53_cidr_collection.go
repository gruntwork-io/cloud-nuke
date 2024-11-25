package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func (r *Route53CidrCollection) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var ids []*string

	result, err := r.Client.ListCidrCollections(r.Context, &route53.ListCidrCollectionsInput{})
	if err != nil {
		logging.Errorf("[Failed] unable to list cidr collection: %s", err)
		return nil, err
	}

	for _, r := range result.CidrCollections {
		if configObj.Route53CIDRCollection.ShouldInclude(config.ResourceValue{
			Name: r.Name,
		}) {
			ids = append(ids, r.Id)
		}
	}

	return ids, nil
}

func (r *Route53CidrCollection) nukeCidrLocations(id *string) (err error) {
	// get attached cidr blocks
	loc, err := r.Client.ListCidrBlocks(r.Context, &route53.ListCidrBlocksInput{
		CollectionId: id,
	})
	if err != nil {
		logging.Errorf("[Failed] unable to list cidr blocks: %v", err)
		return err
	}

	var changes []types.CidrCollectionChange
	for _, block := range loc.CidrBlocks {
		changes = append(changes, types.CidrCollectionChange{
			CidrList:     []string{aws.ToString(block.CidrBlock)},
			Action:       types.CidrCollectionChangeActionDeleteIfExists,
			LocationName: block.LocationName,
		})
	}

	if len(changes) > 0 {
		_, err = r.Client.ChangeCidrCollection(r.Context, &route53.ChangeCidrCollectionInput{
			Id:      id,
			Changes: changes,
		})
		if err != nil {
			logging.Errorf("[Failed] unable to list cidr collections: %v", err)
			return err
		}
	}

	logging.Debugf("[Route53 CIDR location] Successfully nuked cidr location(s)")
	return nil
}

func (r *Route53CidrCollection) nukeAll(identifiers []*string) (err error) {
	if len(identifiers) == 0 {
		logging.Debugf("No Route53 Cidr collection to nuke")
		return nil
	}
	logging.Debugf("Deleting all Route53 Cidr collection")

	var deletedIds []*string
	for _, id := range identifiers {

		err := func() error {

			// remove the cidr blocks
			if err := r.nukeCidrLocations(id); err != nil {
				return err
			}

			// delete the cidr collection
			if _, err = r.Client.DeleteCidrCollection(r.Context, &route53.DeleteCidrCollectionInput{
				Id: id,
			}); err != nil {
				logging.Errorf("[Failed] unable to nuke the cidr collection: %v ", err)
				return err
			}

			return nil
		}()

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "Route53 cidr collection",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *id, err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Route53 cidr collection: %s", aws.ToString(id))
		}
	}

	logging.Debugf("[OK] %d Route53 cidr collection(s) deleted", len(deletedIds))

	return nil
}
