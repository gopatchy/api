package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAPI(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	openapi, err := ta.pyc.OpenAPI(ctx)
	require.NoError(t, err)
	require.NotNil(t, openapi)
	require.NotEmpty(t, openapi.OpenAPI)
}
