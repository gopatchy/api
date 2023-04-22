package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAPI(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	openapi, err := c.OpenAPI(ctx)
	require.NoError(t, err)
	require.NotNil(t, openapi)
	require.IsType(t, "", openapi["openapi"], openapi)
	require.NotEmpty(t, openapi["openapi"].(string))
}
