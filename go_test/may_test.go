package gotest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"test/goclient"
)

func TestMayWriteCreateSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)
}

func TestMayWriteCreateRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetHeader("X-Refuse-Write", "x")

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.Error(t, err)
	require.Nil(t, created)
}

func TestMayWriteReplaceSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	_, err = c.ReplaceMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.NoError(t, err)
}

func TestMayWriteReplaceRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Write", "x")

	_, err = c.ReplaceMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.Error(t, err)
}

func TestMayWriteUpdateSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	_, err = c.UpdateMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.NoError(t, err)
}

func TestMayWriteUpdateRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Write", "x")

	_, err = c.UpdateMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.Error(t, err)
}

func TestMayWriteDeleteSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	err = c.DeleteMayType(ctx, created.ID, nil)
	require.NoError(t, err)
}

func TestMayWriteDeleteRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Write", "x")

	err = c.DeleteMayType(ctx, created.ID, nil)
	require.Error(t, err)
}

func TestMayReadGetSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, get)
}

func TestMayReadGetRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Read", "x")

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.Error(t, err)
	require.Nil(t, get)
}

func TestMayReadStreamGetSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	stream, err := c.StreamGetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	ev := stream.Read()
	require.NotNil(t, ev, stream.Error())
}

func TestMayReadStreamGetRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Read", "x")

	stream, err := c.StreamGetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	ev := stream.Read()
	require.Nil(t, ev)
}

func TestMayReadListSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	list, err := c.ListMayType(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestMayReadListRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Read", "x")

	list, err := c.ListMayType(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestMayReadStreamListSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	stream, err := c.StreamListMayType(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 1)
}

func TestMayReadStreamListRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Read", "x")

	stream, err := c.StreamListMayType(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 0)
}

func TestMayReadCreateSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)
}

func TestMayReadCreateRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetHeader("X-Refuse-Read", "x")

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.Error(t, err)
}

func TestMayReadReplaceSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	_, err = c.ReplaceMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.NoError(t, err)
}

func TestMayReadReplaceRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Read", "x")

	_, err = c.ReplaceMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.Error(t, err)
}

func TestMayReadUpdateSuccess(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	_, err = c.UpdateMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.NoError(t, err)
}

func TestMayReadUpdateRefuse(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{})
	require.NoError(t, err)

	c.SetHeader("X-Refuse-Read", "x")

	_, err = c.UpdateMayType(ctx, created.ID, &goclient.MayTypeRequest{}, nil)
	require.Error(t, err)
}

func TestMayWriteMutateCreate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetHeader("X-Text1-Write", "1234")

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "1234", get.Text1)
}

func TestMayWriteMutateReplace(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Write", "2345")

	_, err = c.ReplaceMayType(ctx, created.ID, &goclient.MayTypeRequest{Text1: "bar"}, nil)
	require.NoError(t, err)

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "2345", get.Text1)
}

func TestMayWriteMutateUpdate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Write", "3456")

	_, err = c.UpdateMayType(ctx, created.ID, &goclient.MayTypeRequest{Text1: "bar"}, nil)
	require.NoError(t, err)

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "3456", get.Text1)
}

func TestMayReadMutateGet(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Read", "1234")

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "1234", get.Text1)
}

func TestMayReadMutateCreate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	c.SetHeader("X-Text1-Read", "2345")

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)
	require.Equal(t, "2345", created.Text1)

	c.SetHeader("X-Text1-Read", "")

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text1)
}

func TestMayReadMutateReplace(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Read", "3456")

	replaced, err := c.ReplaceMayType(ctx, created.ID, &goclient.MayTypeRequest{Text1: "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, "3456", replaced.Text1)

	c.SetHeader("X-Text1-Read", "")

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text1)
}

func TestMayReadMutateUpdate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Read", "4567")

	updated, err := c.UpdateMayType(ctx, created.ID, &goclient.MayTypeRequest{Text1: "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, "4567", updated.Text1)

	c.SetHeader("X-Text1-Read", "")

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text1)
}

func TestMayReadMutateStreamGet(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Read", "5678")

	stream, err := c.StreamGetMayType(ctx, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1)
	require.Equal(t, "5678", s1.Text1)
}

func TestMayReadMutateList(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Read", "6789")

	list, err := c.ListMayType(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "6789", list[0].Text1)
}

func TestMayReadMutateStreamList(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	_, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-Text1-Read", "789a")

	stream, err := c.StreamListMayType(ctx, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "789a", s1[0].Text1)
}

func TestMayReadSideEffect(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateMayType(ctx, &goclient.MayTypeRequest{Text1: "foo"})
	require.NoError(t, err)

	c.SetHeader("X-NewText1", "abcd")

	get, err := c.GetMayType(ctx, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text1)

	c.SetHeader("X-NewText1", "")

	list, err := c.ListMayType(ctx, &goclient.ListOpts{Sorts: []string{"+text1"}})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "abcd", list[0].Text1)
	require.Equal(t, "foo", list[1].Text1)
}
