package resources

import (
	"context"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

// mockCloudSQLClient implements CloudSQLInstancesAPI for testing.
type mockCloudSQLClient struct {
	instances []*sqladmin.DatabaseInstance
	listErr   error
}

func (m *mockCloudSQLClient) ListInstancePages(_ context.Context, _ string, fn func(*sqladmin.InstancesListResponse) error) error {
	if m.listErr != nil {
		return m.listErr
	}
	return fn(&sqladmin.InstancesListResponse{Items: m.instances})
}

func (m *mockCloudSQLClient) DeleteInstance(_ context.Context, _, _ string) (*sqladmin.Operation, error) {
	return &sqladmin.Operation{Name: "op-1", Status: "DONE"}, nil
}

func (m *mockCloudSQLClient) GetOperation(_ context.Context, _, _ string) (*sqladmin.Operation, error) {
	return &sqladmin.Operation{Status: "DONE"}, nil
}

func TestCloudSQLInstances_ResourceName(t *testing.T) {
	t.Parallel()
	cs := NewCloudSQLInstances()
	assert.Equal(t, "cloud-sql-instance", cs.ResourceName())
}

func TestCloudSQLInstances_MaxBatchSize(t *testing.T) {
	t.Parallel()
	cs := NewCloudSQLInstances()
	assert.Equal(t, 50, cs.MaxBatchSize())
}

// TestCloudSQLInstances_ReplicaOrdering verifies that read replicas and read pool
// instances are always returned before primary instances. The Cloud SQL API rejects
// deletion of a primary that still has live replicas, so this ordering is a
// correctness invariant.
func TestCloudSQLInstances_ReplicaOrdering(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	mock := &mockCloudSQLClient{
		instances: []*sqladmin.DatabaseInstance{
			{Name: "primary-1", InstanceType: "CLOUD_SQL_INSTANCE", CreateTime: now, State: "RUNNABLE"},
			{Name: "replica-1", InstanceType: "READ_REPLICA_INSTANCE", CreateTime: now, State: "RUNNABLE"},
			{Name: "primary-2", InstanceType: "CLOUD_SQL_INSTANCE", CreateTime: now, State: "RUNNABLE"},
			{Name: "replica-2", InstanceType: "READ_REPLICA_INSTANCE", CreateTime: now, State: "RUNNABLE"},
		},
	}

	ids, err := listCloudSQLInstances(context.Background(), mock, resource.Scope{ProjectID: "my-project"}, config.ResourceType{})
	require.NoError(t, err)
	require.Len(t, ids, 4)

	var names []string
	for _, id := range ids {
		names = append(names, *id)
	}

	// Replicas must come before primaries regardless of the order returned by the API.
	assert.Equal(t, []string{
		"my-project/replica-1",
		"my-project/replica-2",
		"my-project/primary-1",
		"my-project/primary-2",
	}, names)
}
