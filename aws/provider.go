package aws

import (
	"context"
	"sync"

	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// provider adapts AWS to the provider-neutral engine in the resource package.
// It holds only what the engine cannot know: which regions to walk, how to
// build a session for one, and which failures are rate limits.
type provider struct {
	regions []string

	// setAccountIdOnce keeps telemetry's account id from being written by every
	// region at once; every region reports the same account.
	setAccountIdOnce sync.Once
}

// Targets returns one target per region, with the global pseudo-region deferred
// to the end: IAM, S3, CloudFront and Route53 deletions can break a regional
// delete that is still in flight.
func (p *provider) Targets() ([]resource.Target, error) {
	targets := make([]resource.Target, 0, len(p.regions))
	for _, region := range p.regions {
		targets = append(targets, resource.Target{
			ID:   region,
			Last: region == GlobalRegion,
		})
	}
	return targets, nil
}

// Prepare builds a session for one region and attaches the account id, which
// resources read from the context when they tag what they have seen.
func (p *provider) Prepare(ctx context.Context, target resource.Target) (context.Context, []resource.NukeableResource, error) {
	session, err := NewSession(target.ID)
	if err != nil {
		return nil, nil, err
	}

	if accountID, err := util.GetCurrentAccountId(session); err == nil {
		p.setAccountIdOnce.Do(func() { telemetry.SetAccountId(accountID) })
		ctx = context.WithValue(ctx, util.AccountIdKey, accountID)
	}

	registered := GetAndInitRegisteredResources(session, target.ID)
	nukeable := make([]resource.NukeableResource, len(registered))
	for i := range registered {
		nukeable[i] = *registered[i]
	}
	return ctx, nukeable, nil
}

// SkipListError reports that AWS passes over nothing: every listing failure is
// shown, because an AWS API answering with an error is not routine.
func (p *provider) SkipListError(err error, resourceType string, explicitlyRequested bool) bool {
	return false
}

// IsThrottle reports whether the error is an AWS rate limit.
func (p *provider) IsThrottle(err error) bool {
	return util.IsThrottlingError(err)
}

var _ resource.Provider = (*provider)(nil)
