package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (rdp *RdsProxy) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var names []*string
	paginator := rds.NewDescribeDBProxiesPaginator(rdp.Client, &rds.DescribeDBProxiesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(rdp.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, proxy := range page.DBProxies {
			if configObj.RdsProxy.ShouldInclude(config.ResourceValue{
				Name: proxy.DBProxyName,
				Time: proxy.CreatedDate,
			}) {
				names = append(names, proxy.DBProxyName)
			}
		}
	}

	return names, nil
}
func (rdp *RdsProxy) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No RDS proxy in region %s", rdp.Region)
		return nil
	}

	logging.Debugf("Deleting all DB Proxies in region %s", rdp.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		logging.Debugf("[RDS Proxy] Deleting %s in region %s", *identifier, rdp.Region)

		_, err := rdp.Client.DeleteDBProxy(
			rdp.Context,
			&rds.DeleteDBProxyInput{
				DBProxyName: identifier,
			})
		if err != nil {
			logging.Errorf("[RDS Proxy] Error deleting RDS Proxy %s: %s", *identifier, err)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[RDS Proxy] Deleted RDS Proxy %s", *identifier)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: rdp.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d RDS DB proxi(s) nuked in %s", len(deleted), rdp.Region)
	return nil
}
