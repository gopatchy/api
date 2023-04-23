package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBearerAuthSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetAuthToken("abcd")

	list, err := c.ListAuthBearerType(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)
}

func TestBearerAuthInvalidToken(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetAuthToken("bcde")

	_, err := c.ListAuthBearerType(ctx, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "token not found")
}

func TestBearerAuthOptional(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.ListAuthBearerType(ctx, nil)
	require.NoError(t, err)
}
