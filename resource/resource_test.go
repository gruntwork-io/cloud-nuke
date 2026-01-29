package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct{}

func TestResource_MaxBatchSize(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		r := &Resource[*mockClient]{}
		assert.Equal(t, DefaultBatchSize, r.MaxBatchSize())
	})

	t.Run("custom", func(t *testing.T) {
		r := &Resource[*mockClient]{BatchSize: 50}
		assert.Equal(t, 50, r.MaxBatchSize())
	})
}

func TestResource_Init(t *testing.T) {
	initCalled := false
	r := &Resource[*mockClient]{
		InitClient: func(r *Resource[*mockClient], cfg any) {
			initCalled = true
			r.Client = &mockClient{}
			r.Scope.Region = "us-east-1"
		},
	}

	r.Init("test-config")

	assert.True(t, initCalled)
	assert.NotNil(t, r.Client)
	assert.Equal(t, "us-east-1", r.Scope.Region)
}

func TestResource_GetAndSetIdentifiers(t *testing.T) {
	r := &Resource[*mockClient]{
		ResourceTypeName: "test",
		Lister: func(ctx context.Context, client *mockClient, scope Scope, resourceCfg config.ResourceType) ([]*string, error) {
			return []*string{aws.String("id-1"), aws.String("id-2")}, nil
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return config.ResourceType{}
		},
	}
	r.Init(nil)

	ids, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})

	require.NoError(t, err)
	assert.Equal(t, []string{"id-1", "id-2"}, ids)
	assert.Equal(t, []string{"id-1", "id-2"}, r.ResourceIdentifiers())
}

func TestResource_GetAndSetIdentifiers_MissingConfig(t *testing.T) {
	r := &Resource[*mockClient]{ResourceTypeName: "test"}

	_, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestResource_Nuke(t *testing.T) {
	nuked := []string{}
	r := &Resource[*mockClient]{
		ResourceTypeName: "test",
		Nuker: func(ctx context.Context, client *mockClient, scope Scope, resourceType string, ids []*string) []NukeResult {
			results := make([]NukeResult, len(ids))
			for i, id := range ids {
				nuked = append(nuked, *id)
				results[i] = NukeResult{Identifier: *id, Error: nil}
			}
			return results
		},
	}
	r.Init(nil)

	results, err := r.Nuke(context.Background(), []string{"id-1", "id-2"})

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, []string{"id-1", "id-2"}, nuked)
}

func TestResource_Nuke_EmptySkipsNuker(t *testing.T) {
	nukerCalled := false
	r := &Resource[*mockClient]{
		Nuker: func(ctx context.Context, client *mockClient, scope Scope, resourceType string, ids []*string) []NukeResult {
			nukerCalled = true
			return nil
		},
	}

	results, err := r.Nuke(context.Background(), []string{})

	require.NoError(t, err)
	assert.Nil(t, results)
	assert.False(t, nukerCalled)
}

func TestResource_Nuke_PropagatesError(t *testing.T) {
	r := &Resource[*mockClient]{
		ResourceTypeName: "test",
		Nuker: func(ctx context.Context, client *mockClient, scope Scope, resourceType string, ids []*string) []NukeResult {
			return []NukeResult{{Identifier: *ids[0], Error: errors.New("delete failed")}}
		},
	}
	r.Init(nil)

	results, err := r.Nuke(context.Background(), []string{"id-1"})

	require.Error(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, err.Error(), "delete failed")
}

func TestResource_PermissionVerification(t *testing.T) {
	r := &Resource[*mockClient]{
		ResourceTypeName: "test",
		Lister: func(ctx context.Context, client *mockClient, scope Scope, resourceCfg config.ResourceType) ([]*string, error) {
			return []*string{aws.String("allowed"), aws.String("denied")}, nil
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return config.ResourceType{}
		},
		PermissionVerifier: func(ctx context.Context, client *mockClient, id *string) error {
			if *id == "denied" {
				return errors.New("access denied")
			}
			return nil
		},
	}
	r.Init(nil)

	_, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)

	nukable, _ := r.IsNukable("allowed")
	assert.True(t, nukable)

	nukable, err = r.IsNukable("denied")
	assert.False(t, nukable)
	assert.Contains(t, err.Error(), "access denied")
}

func TestResource_IsNukable(t *testing.T) {
	r := &Resource[*mockClient]{}
	r.Init(nil)

	// Unknown = nukable by default
	nukable, err := r.IsNukable("unknown")
	assert.True(t, nukable)
	assert.NoError(t, err)

	// Explicitly set as not nukable
	r.setNukableStatus("bad", errors.New("cannot nuke"))
	nukable, err = r.IsNukable("bad")
	assert.False(t, nukable)
	assert.Error(t, err)
}

// Batch Deleter Tests

func TestSimpleBatchDeleter(t *testing.T) {
	deleteCount := 0
	deleter := SimpleBatchDeleter(func(ctx context.Context, client *mockClient, id *string) error {
		deleteCount++
		return nil
	})

	ids := []*string{aws.String("1"), aws.String("2"), aws.String("3")}
	results := deleter(context.Background(), &mockClient{}, Scope{}, "test", ids)

	assert.Len(t, results, 3)
	for _, result := range results {
		assert.NoError(t, result.Error)
	}
	assert.Equal(t, 3, deleteCount)
}

func TestSequentialDeleter(t *testing.T) {
	order := []string{}
	deleter := SequentialDeleter(func(ctx context.Context, client *mockClient, id *string) error {
		order = append(order, *id)
		return nil
	})

	ids := []*string{aws.String("a"), aws.String("b"), aws.String("c")}
	results := deleter(context.Background(), &mockClient{}, Scope{}, "test", ids)

	assert.Len(t, results, 3)
	for _, result := range results {
		assert.NoError(t, result.Error)
	}
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestSequentialDeleter_AccumulatesErrors(t *testing.T) {
	deleter := SequentialDeleter(func(ctx context.Context, client *mockClient, id *string) error {
		if *id == "fail" {
			return errors.New("failed")
		}
		return nil
	})

	ids := []*string{aws.String("ok"), aws.String("fail"), aws.String("also-ok")}
	results := deleter(context.Background(), &mockClient{}, Scope{}, "test", ids)

	assert.Len(t, results, 3)
	assert.NoError(t, results[0].Error)
	assert.Error(t, results[1].Error)
	assert.Contains(t, results[1].Error.Error(), "failed")
	assert.NoError(t, results[2].Error)
}

func TestMultiStepDeleter(t *testing.T) {
	steps := []string{}
	deleter := MultiStepDeleter(
		func(ctx context.Context, client *mockClient, id *string) error {
			steps = append(steps, "step1")
			return nil
		},
		func(ctx context.Context, client *mockClient, id *string) error {
			steps = append(steps, "step2")
			return nil
		},
	)

	results := deleter(context.Background(), &mockClient{}, Scope{}, "test", []*string{aws.String("x")})

	assert.Len(t, results, 1)
	assert.NoError(t, results[0].Error)
	assert.Equal(t, []string{"step1", "step2"}, steps)
}

func TestMultiStepDeleter_StopsOnFailure(t *testing.T) {
	step2Called := false
	deleter := MultiStepDeleter(
		func(ctx context.Context, client *mockClient, id *string) error {
			return errors.New("step1 failed")
		},
		func(ctx context.Context, client *mockClient, id *string) error {
			step2Called = true
			return nil
		},
	)

	results := deleter(context.Background(), &mockClient{}, Scope{}, "test", []*string{aws.String("x")})

	assert.Len(t, results, 1)
	assert.Error(t, results[0].Error)
	assert.False(t, step2Called)
}

func TestScope_String(t *testing.T) {
	assert.Equal(t, "us-east-1", Scope{Region: "us-east-1"}.String())
	assert.Equal(t, "my-project", Scope{ProjectID: "my-project"}.String())
	assert.Equal(t, "my-project/us-central1", Scope{ProjectID: "my-project", Region: "us-central1"}.String())
}
