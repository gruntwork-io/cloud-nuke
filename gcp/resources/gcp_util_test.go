package resources

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
)

func TestIsGCPNotFound(t *testing.T) {
	t.Parallel()
	t.Run("404 error returns true", func(t *testing.T) {
		t.Parallel()
		err := &googleapi.Error{Code: 404, Message: "not found"}
		assert.True(t, isGCPNotFound(err))
	})
	t.Run("403 error returns false", func(t *testing.T) {
		t.Parallel()
		err := &googleapi.Error{Code: 403, Message: "forbidden"}
		assert.False(t, isGCPNotFound(err))
	})
	t.Run("non-googleapi error returns false", func(t *testing.T) {
		t.Parallel()
		err := assert.AnError
		assert.False(t, isGCPNotFound(err))
	})
	t.Run("wrapped 404 error returns true", func(t *testing.T) {
		t.Parallel()
		inner := &googleapi.Error{Code: 404, Message: "not found"}
		err := fmt.Errorf("outer context: %w", inner)
		assert.True(t, isGCPNotFound(err))
	})
	t.Run("nil error returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isGCPNotFound(nil))
	})
}
