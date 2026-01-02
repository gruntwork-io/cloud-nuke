package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedCloudMapNamespacesAPI struct {
	CloudMapNamespacesAPI
	ListNamespacesOutput      servicediscovery.ListNamespacesOutput
	DeleteNamespaceOutput     servicediscovery.DeleteNamespaceOutput
	GetNamespaceOutput        servicediscovery.GetNamespaceOutput
	ListTagsForResourceOutput servicediscovery.ListTagsForResourceOutput
	TagsByArn                 map[string][]types.Tag
}

func (m mockedCloudMapNamespacesAPI) ListNamespaces(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
	return &m.ListNamespacesOutput, nil
}

func (m mockedCloudMapNamespacesAPI) DeleteNamespace(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error) {
	return &m.DeleteNamespaceOutput, nil
}

func (m mockedCloudMapNamespacesAPI) GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error) {
	return &m.GetNamespaceOutput, nil
}

func (m mockedCloudMapNamespacesAPI) ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
	if m.TagsByArn != nil {
		if tags, ok := m.TagsByArn[aws.ToString(params.ResourceARN)]; ok {
			return &servicediscovery.ListTagsForResourceOutput{Tags: tags}, nil
		}
	}
	return &m.ListTagsForResourceOutput, nil
}

func TestCloudMapNamespaces_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ns1 := types.NamespaceSummary{
		Id:         aws.String("ns-123456789"),
		Arn:        aws.String("arn:aws:servicediscovery:us-east-1:123456789012:namespace/ns-123456789"),
		Name:       aws.String("test-namespace-1"),
		CreateDate: aws.Time(now.Add(-1 * time.Hour)),
	}
	ns2 := types.NamespaceSummary{
		Id:         aws.String("ns-987654321"),
		Arn:        aws.String("arn:aws:servicediscovery:us-east-1:123456789012:namespace/ns-987654321"),
		Name:       aws.String("test-namespace-2"),
		CreateDate: aws.Time(now),
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"ns-123456789", "ns-987654321"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-namespace-1")}},
				},
			},
			expected: []string{"ns-987654321"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-30 * time.Minute)),
				},
			},
			expected: []string{"ns-123456789"},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{"ns-987654321"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockedCloudMapNamespacesAPI{
				ListNamespacesOutput: servicediscovery.ListNamespacesOutput{
					Namespaces: []types.NamespaceSummary{ns1, ns2},
				},
				TagsByArn: map[string][]types.Tag{
					aws.ToString(ns1.Arn): {{Key: aws.String("Environment"), Value: aws.String("test")}},
					aws.ToString(ns2.Arn): {{Key: aws.String("Environment"), Value: aws.String("production")}},
				},
			}

			ids, err := listCloudMapNamespaces(context.Background(), client, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestCloudMapNamespaces_Nuke(t *testing.T) {
	t.Parallel()

	client := mockedCloudMapNamespacesAPI{
		GetNamespaceOutput: servicediscovery.GetNamespaceOutput{
			Namespace: &types.Namespace{Id: aws.String("ns-123456789"), Name: aws.String("test-namespace")},
		},
		DeleteNamespaceOutput: servicediscovery.DeleteNamespaceOutput{OperationId: aws.String("operation-123")},
	}

	err := deleteCloudMapNamespace(context.Background(), client, aws.String("ns-123456789"))
	require.NoError(t, err)
}
