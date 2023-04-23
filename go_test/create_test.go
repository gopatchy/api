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

func TestCreateB(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateTestType(ctx, &goclient.TestType{Text: "foo"})
	require.NoError(t, err)

	_, err = c.CreateTestTypeB(ctx, &goclient.TestType{Text: "bar"})
	require.NoError(t, err)

	list1, err := c.ListTestType(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list1, 1)
	require.Equal(t, "foo", list1[0].Text)

	list2, err := c.ListTestTypeB(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list2, 1)
	require.Equal(t, "bar", list2[0].Text)
}
