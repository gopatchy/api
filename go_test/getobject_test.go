package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"test/goclient"
)

func TestGetPrev(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.EqualValues(t, 0, get.Num)

	// Validate that previous version passing only compares the ETag
	get.Num = 1

	get2, err := c.GetTestType(ctx, created.ID, &goclient.GetOpts{Prev: get})
	require.NoError(t, err)
	require.Equal(t, "foo", get2.Text)
	require.EqualValues(t, 1, get2.Num)
}
