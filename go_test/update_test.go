package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"test/goclient"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestType{Text: "foo", Num: 1})
	require.NoError(t, err)

	updated, err := c.UpdateTestType(ctx, created.ID, &goclient.TestType{Text: "bar"}, nil)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "bar", updated.Text)
	require.EqualValues(t, 1, updated.Num)
	require.EqualValues(t, created.Generation+1, updated.Generation)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
	require.EqualValues(t, 1, get.Num)
	require.EqualValues(t, created.Generation+1, updated.Generation)
}

func TestUpdateNotExist(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	updated, err := c.UpdateTestType(ctx, "doesnotexist", &goclient.TestType{Text: "bar"}, nil)
	require.Error(t, err)
	require.Nil(t, updated)
}

func TestUpdateIfMatchETagSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestType{Text: "foo"})
	require.NoError(t, err)

	updated, err := c.UpdateTestType(ctx, created.ID, &goclient.TestType{Text: "bar"}, &goclient.UpdateOpts[goclient.TestType]{Prev: created})
	require.NoError(t, err)
	require.Equal(t, "bar", updated.Text)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
}

func TestUpdateIfMatchETagMismatch(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestType{Text: "foo"})
	require.NoError(t, err)

	created.ETag = "etag:doesnotmatch"

	updated, err := c.UpdateTestType(ctx, created.ID, &goclient.TestType{Text: "bar"}, &goclient.UpdateOpts[goclient.TestType]{Prev: created})
	require.Error(t, err)
	require.Nil(t, updated)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
}
