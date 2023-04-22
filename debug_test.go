package patchy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDebugInfo(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	debug, err := ta.pyc.DebugInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, debug)
	require.IsType(t, debug["server"], map[string]any{})
	require.IsType(t, debug["server"].(map[string]any)["hostname"], "")
	require.NotEmpty(t, debug["server"].(map[string]any)["hostname"].(string))
}
