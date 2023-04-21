package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"test/goclient"
)

func TestReplace(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo", Num: 1})
	require.NoError(t, err)

	replaced, err := c.ReplaceTestType(ctx, created.ID, &goclient.TestTypeRequest{Text: "bar"}, nil)
	require.NoError(t, err)
	require.NotNil(t, replaced)
	require.Equal(t, "bar", replaced.Text)
	require.EqualValues(t, 0, replaced.Num)
	require.EqualValues(t, created.Generation+1, replaced.Generation)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
	require.EqualValues(t, 0, get.Num)
	require.EqualValues(t, created.Generation+1, get.Generation)
}

func TestReplaceNotExist(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	replaced, err := c.ReplaceTestType(ctx, "doesnotexist", &goclient.TestTypeRequest{Text: "bar"}, nil)
	require.Error(t, err)
	require.Nil(t, replaced)
}

func TestReplaceIfMatchETagSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	replaced, err := c.ReplaceTestType(ctx, created.ID, &goclient.TestTypeRequest{Text: "bar"}, &goclient.UpdateOpts{Prev: created})
	require.NoError(t, err)
	require.Equal(t, "bar", replaced.Text)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
}

func TestReplaceIfMatchETagMismatch(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	created.ETag = "etag:doesnotmatch"

	replaced, err := c.ReplaceTestType(ctx, created.ID, &goclient.TestTypeRequest{Text: "bar"}, &goclient.UpdateOpts{Prev: created})
	require.Error(t, err)
	require.Nil(t, replaced)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
}
