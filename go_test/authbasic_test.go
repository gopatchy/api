package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicAuthSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetBasicAuth("foo", "abcd")

	list, err := c.ListAuthBasicType(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)
}

func TestBasicAuthInvalidUser(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetBasicAuth("bar", "abcd")

	_, err := c.ListAuthBasicType(ctx, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "user not found")
}

func TestBasicAuthInvalidPassword(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetBasicAuth("foo", "bcde")

	_, err := c.ListAuthBasicType(ctx, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "user password mismatch")
}

func TestBasicAuthOptional(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.ListAuthBasicType(ctx, nil)
	require.NoError(t, err)
}
