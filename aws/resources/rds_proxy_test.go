package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedRdsProxy struct {
	rdsiface.RDSAPI
	DescribeDBProxiesOutput rds.DescribeDBProxiesOutput
	DeleteDBProxyOutput     rds.DeleteDBProxyOutput
}

func (m mockedRdsProxy) DescribeDBProxiesPagesWithContext(_ aws.Context, _ *rds.DescribeDBProxiesInput, callback func(*rds.DescribeDBProxiesOutput, bool) bool, _ ...request.Option) error {
	callback(&m.DescribeDBProxiesOutput, true)
	return nil
}

func (m mockedRdsProxy) DeleteDBProxyWithContext(aws.Context, *rds.DeleteDBProxyInput, ...request.Option) (*rds.DeleteDBProxyOutput, error) {
	return &m.DeleteDBProxyOutput, nil
}

func TestRdsProxy_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	snapshot := RdsProxy{
		Client: mockedRdsProxy{
			DescribeDBProxiesOutput: rds.DescribeDBProxiesOutput{
				DBProxies: []*rds.DBProxy{
					{
						DBProxyName: &testName1,
						CreatedDate: &now,
					},
					{
						DBProxyName: &testName2,
						CreatedDate: aws.Time(now.Add(1)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := snapshot.getAll(context.Background(), config.Config{
				RdsProxy: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestRdsProxy_NukeAll(t *testing.T) {

	t.Parallel()

	testName := "test-db-proxy"
	snapshot := RdsProxy{
		Client: mockedRdsProxy{
			DeleteDBProxyOutput: rds.DeleteDBProxyOutput{},
		},
	}

	err := snapshot.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
