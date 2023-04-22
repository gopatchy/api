package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDebugInfo(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	debug, err := c.DebugInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, debug)
	require.IsType(t, map[string]any{}, debug["server"])
	require.IsType(t, "", debug["server"].(map[string]any)["hostname"])
	require.NotEmpty(t, debug["server"].(map[string]any)["hostname"].(string))
}
