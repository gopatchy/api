package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"test/goclient"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestType{Text: "foo"})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "foo", created.Text)
	require.NotEmpty(t, created.ID)

	get, err := c.GetTestType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.Equal(t, created.ID, get.ID)
}
