package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockNetworkInterfaceClient struct {
	DescribeOutput     ec2.DescribeNetworkInterfacesOutput
	DeleteOutput       ec2.DeleteNetworkInterfaceOutput
	DescribeAddrOutput ec2.DescribeAddressesOutput
	TerminateOutput    ec2.TerminateInstancesOutput
	ReleaseOutput      ec2.ReleaseAddressOutput
	DescribeVpcsOutput ec2.DescribeVpcsOutput
	DescribeError      error
	DeleteError        error
	DescribeAddrError  error
	TerminateError     error
}

func (m *mockNetworkInterfaceClient) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &m.DescribeOutput, m.DescribeError
}

func (m *mockNetworkInterfaceClient) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	return &m.DeleteOutput, m.DeleteError
}

func (m *mockNetworkInterfaceClient) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddrOutput, m.DescribeAddrError
}

func (m *mockNetworkInterfaceClient) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseOutput, nil
}

func (m *mockNetworkInterfaceClient) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateOutput, m.TerminateError
}

func (m *mockNetworkInterfaceClient) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{}, nil
}

func (m *mockNetworkInterfaceClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func (m *mockNetworkInterfaceClient) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func TestListNetworkInterfaces(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name     string
		mock     *mockNetworkInterfaceClient
		cfg      config.ResourceType
		expected []string
	}{
		{
			name: "lists all interfaces with no filter",
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{
							NetworkInterfaceId: aws.String("eni-001"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("interface1")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
						{
							NetworkInterfaceId: aws.String("eni-002"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("interface2")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
					},
				},
			},
			cfg:      config.ResourceType{},
			expected: []string{"eni-001", "eni-002"},
		},
		{
			name: "excludes by name regex",
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{
							NetworkInterfaceId: aws.String("eni-001"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("keep-this")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
						{
							NetworkInterfaceId: aws.String("eni-002"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("skip-this")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
					},
				},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
				},
			},
			expected: []string{"eni-001"},
		},
		{
			name: "includes by name regex",
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{
							NetworkInterfaceId: aws.String("eni-001"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("keep-this")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
						{
							NetworkInterfaceId: aws.String("eni-002"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("skip-this")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
					},
				},
			},
			cfg: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("keep-.*")}},
				},
			},
			expected: []string{"eni-001"},
		},
		{
			name: "skips non-interface types",
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{
							NetworkInterfaceId: aws.String("eni-001"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("interface1")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
						{
							NetworkInterfaceId: aws.String("eni-002"),
							InterfaceType:      "lambda",
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("lambda-eni")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
					},
				},
			},
			cfg:      config.ResourceType{},
			expected: []string{"eni-001"},
		},
		{
			name: "excludes by time after",
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{
							NetworkInterfaceId: aws.String("eni-001"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("interface1")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
							},
						},
						{
							NetworkInterfaceId: aws.String("eni-002"),
							InterfaceType:      NetworkInterfaceTypeInterface,
							TagSet: []types.Tag{
								{Key: aws.String("Name"), Value: aws.String("interface2")},
								{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
							},
						},
					},
				},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{"eni-001"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
			ids, err := listNetworkInterfaces(ctx, tc.mock, resource.Scope{}, tc.cfg, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteNetworkInterfaceByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    *mockNetworkInterfaceClient
		wantErr bool
	}{
		{
			name:    "successful delete",
			mock:    &mockNetworkInterfaceClient{},
			wantErr: false,
		},
		{
			name: "no attachment - successful delete",
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{
							NetworkInterfaceId: aws.String("eni-test"),
							// No Attachment field
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := deleteNetworkInterfaceByID(context.Background(), tc.mock, aws.String("eni-test"))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDetachNetworkInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      *string
		mock    *mockNetworkInterfaceClient
		wantErr bool
	}{
		{
			name:    "nil id returns nil",
			id:      nil,
			mock:    &mockNetworkInterfaceClient{},
			wantErr: false,
		},
		{
			name:    "empty id returns nil",
			id:      aws.String(""),
			mock:    &mockNetworkInterfaceClient{},
			wantErr: false,
		},
		{
			name: "no attachment skips detachment",
			id:   aws.String("eni-test"),
			mock: &mockNetworkInterfaceClient{
				DescribeOutput: ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{NetworkInterfaceId: aws.String("eni-test")},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := detachNetworkInterface(context.Background(), tc.mock, tc.id)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReleaseNetworkInterfaceEIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		instanceID *string
		mock       *mockNetworkInterfaceClient
		wantErr    bool
	}{
		{
			name:       "nil instance id returns nil",
			instanceID: nil,
			mock:       &mockNetworkInterfaceClient{},
			wantErr:    false,
		},
		{
			name:       "empty instance id returns nil",
			instanceID: aws.String(""),
			mock:       &mockNetworkInterfaceClient{},
			wantErr:    false,
		},
		{
			name:       "no addresses to release",
			instanceID: aws.String("i-12345"),
			mock: &mockNetworkInterfaceClient{
				DescribeAddrOutput: ec2.DescribeAddressesOutput{
					Addresses: []types.Address{},
				},
			},
			wantErr: false,
		},
		{
			name:       "releases addresses successfully",
			instanceID: aws.String("i-12345"),
			mock: &mockNetworkInterfaceClient{
				DescribeAddrOutput: ec2.DescribeAddressesOutput{
					Addresses: []types.Address{
						{AllocationId: aws.String("eipalloc-001")},
						{AllocationId: aws.String("eipalloc-002")},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := releaseNetworkInterfaceEIPs(context.Background(), tc.mock, tc.instanceID)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
