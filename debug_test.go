package api_test

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
	require.NotEmpty(t, debug.Server.Hostname)
}
