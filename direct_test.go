package patchy_test

import (
	"context"
	"testing"
	"time"

	"github.com/gopatchy/patchy"
	"github.com/stretchr/testify/require"
)

func TestDirectGetNotFound(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	get, err := patchy.Get[testType](ctx, ta.api, "doesnotexist", nil)
	require.NoError(t, err)
	require.Nil(t, get)
}

func TestDirectGetInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.GetName[testType](ctx, ta.api, "doesnotexist", create.ID, nil)
	require.Error(t, err)
}

func TestDirectCreateGet(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)
	require.Equal(t, "foo", create.Text)

	get, err := patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Equal(t, create.ID, get.ID)
	require.Equal(t, "foo", get.Text)
}

func TestDirectCreateInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.CreateName[testType](ctx, ta.api, "doesnotexist", &testType{Text: "foo"})
	require.Error(t, err)
}

func TestDirectUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo", Num: 1})
	require.NoError(t, err)

	get, err := patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.EqualValues(t, 1, get.Num)

	update, err := patchy.UpdateMap[testType](ctx, ta.api, create.ID, map[string]any{"text": "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, create.ID, update.ID)
	require.Equal(t, "bar", update.Text)
	require.EqualValues(t, 1, update.Num)

	get, err = patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
	require.EqualValues(t, 1, get.Num)
}

func TestDirectUpdateInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.UpdateName[testType](ctx, ta.api, "doesnotexist", create.ID, &testType{Text: "bar"}, nil)
	require.Error(t, err)
}

func TestDirectReplace(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo", Num: 1})
	require.NoError(t, err)

	get, err := patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.EqualValues(t, 1, get.Num)

	replace, err := patchy.Replace[testType](ctx, ta.api, create.ID, &testType{Text: "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, create.ID, replace.ID)
	require.Equal(t, "bar", replace.Text)
	require.EqualValues(t, 0, replace.Num)

	get, err = patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
	require.EqualValues(t, 0, get.Num)
}

func TestDirectReplaceInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo", Num: 1})
	require.NoError(t, err)

	get, err := patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.EqualValues(t, 1, get.Num)

	_, err = patchy.ReplaceName[testType](ctx, ta.api, "doesnotexist", create.ID, &testType{Text: "bar"}, nil)
	require.Error(t, err)
}

func TestDirectDelete(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)

	err = patchy.Delete[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)

	get, err := patchy.Get[testType](ctx, ta.api, create.ID, nil)
	require.NoError(t, err)
	require.Nil(t, get)
}

func TestDirectDeleteInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	err = patchy.DeleteName[testType](ctx, ta.api, "doesnotexist", create.ID, nil)
	require.Error(t, err)
}

func TestDirectDeleteNotFound(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	err := patchy.Delete[testType](ctx, ta.api, "doesnotexist", nil)
	require.Error(t, err)
}

func TestDirectList(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.Create[testType](ctx, ta.api, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchy.List[testType](ctx, ta.api, nil)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.ElementsMatch(t, []string{"foo", "bar"}, []string{list[0].Text, list[1].Text})
}

func TestDirectListInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.ListName[testType](ctx, ta.api, "doesnotexist", nil)
	require.Error(t, err)
}

func TestDirectFind(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	find, err := patchy.Find[testType](ctx, ta.api, create.ID[:4])
	require.NoError(t, err)
	require.Equal(t, create.ID, find.ID)
	require.Equal(t, "foo", find.Text)
}

func TestDirectFindNotExist(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	find, err := patchy.Find[testType](ctx, ta.api, "doesnotexist")
	require.Error(t, err)
	require.Nil(t, find)
}

func TestDirectFindMultiple(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.Create[testType](ctx, ta.api, &testType{Text: "bar"})
	require.NoError(t, err)

	find, err := patchy.Find[testType](ctx, ta.api, "")
	require.Error(t, err)
	require.Nil(t, find)
}

func TestDirectStreamGetNotFound(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	stream, err := patchy.StreamGet[testType](ctx, ta.api, "junk")
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()
}

func TestDirectStreamGetInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.StreamGetName[testType](ctx, ta.api, "doesnotexist", create.ID)
	require.Error(t, err)
}

func TestDirectStreamGetInitial(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchy.StreamGet[testType](ctx, ta.api, create.ID)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)
}

func TestDirectStreamGetUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	create, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchy.StreamGet[testType](ctx, ta.api, create.ID)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)

	_, err = patchy.Update[testType](ctx, ta.api, create.ID, &testType{Text: "bar"}, nil)
	require.NoError(t, err)

	s2 := stream.Read()
	require.NotNil(t, s2, stream.Error())
	require.Equal(t, "bar", s2.Text)
}

func TestDirectStreamListInvalidType(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	stream, err := patchy.StreamListName[testType](ctx, ta.api, "invalid", nil)
	require.Error(t, err)
	require.Nil(t, stream)
}

func TestDirectStreamListInitial(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.Create[testType](ctx, ta.api, &testType{Text: "bar"})
	require.NoError(t, err)

	stream, err := patchy.StreamList[testType](ctx, ta.api, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 2)
	require.ElementsMatch(t, []string{"foo", "bar"}, []string{s1[0].Text, s1[1].Text})
}

func TestDirectStreamListUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	stream, err := patchy.StreamList[testType](ctx, ta.api, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 0)

	_, err = patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	s2 := stream.Read()
	require.NotNil(t, s2, stream.Error())
	require.Len(t, s2, 1)
	require.Equal(t, "foo", s2[0].Text)
}

func TestDirectStreamListDelete(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchy.StreamList[testType](ctx, ta.api, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 1)
	require.Equal(t, "foo", s1[0].Text)

	err = patchy.Delete[testType](ctx, ta.api, created.ID, nil)
	require.NoError(t, err)

	s2 := stream.Read()
	require.NotNil(t, s2, stream.Error())
	require.Len(t, s2, 0)
}

func TestDirectStreamListOpts(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchy.Create[testType](ctx, ta.api, &testType{Text: "bar"})
	require.NoError(t, err)

	stream, err := patchy.StreamList[testType](ctx, ta.api, &patchy.ListOpts{Limit: 1})
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 1)
	require.Contains(t, []string{"foo", "bar"}, s1[0].Text)
}

func TestReplicateCreate(t *testing.T) {
	t.Parallel()

	taSrc := newTestAPI(t)
	defer taSrc.shutdown(t)

	taDst := newTestAPI(t)
	defer taDst.shutdown(t)

	ctx := context.Background()

	stream, err := patchy.StreamList[testType](ctx, taSrc.api, nil)
	require.NoError(t, err)

	defer stream.Close()

	go func() {
		err := patchy.ReplicateIn(ctx, taDst.api, stream.Chan(), func(tt *testType) (*testType, error) { return tt, nil }, nil)
		require.NoError(t, err)
	}()

	created, err := patchy.Create(ctx, taSrc.api, &testType{Text: "foo"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	list, err := patchy.List[testType](ctx, taDst.api, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.ID, list[0].ID)
	require.Equal(t, "foo", list[0].Text)
}

func TestReplicateUpdate(t *testing.T) {
	t.Parallel()

	taSrc := newTestAPI(t)
	defer taSrc.shutdown(t)

	taDst := newTestAPI(t)
	defer taDst.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create(ctx, taSrc.api, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchy.StreamList[testType](ctx, taSrc.api, nil)
	require.NoError(t, err)

	defer stream.Close()

	go func() {
		err := patchy.ReplicateIn(ctx, taDst.api, stream.Chan(), func(tt *testType) (*testType, error) { return tt, nil }, nil)
		require.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	list, err := patchy.List[testType](ctx, taDst.api, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.ID, list[0].ID)
	require.Equal(t, "foo", list[0].Text)

	_, err = patchy.Update(ctx, taSrc.api, created.ID, &testType{Text: "bar"}, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	list, err = patchy.List[testType](ctx, taDst.api, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.ID, list[0].ID)
	require.Equal(t, "bar", list[0].Text)
}

func TestReplicateDelete(t *testing.T) {
	t.Parallel()

	taSrc := newTestAPI(t)
	defer taSrc.shutdown(t)

	taDst := newTestAPI(t)
	defer taDst.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create(ctx, taSrc.api, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchy.StreamList[testType](ctx, taSrc.api, nil)
	require.NoError(t, err)

	defer stream.Close()

	go func() {
		err := patchy.ReplicateIn(ctx, taDst.api, stream.Chan(), func(tt *testType) (*testType, error) { return tt, nil }, nil)
		require.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	list, err := patchy.List[testType](ctx, taDst.api, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.ID, list[0].ID)
	require.Equal(t, "foo", list[0].Text)

	err = patchy.Delete[testType](ctx, taSrc.api, created.ID, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	list, err = patchy.List[testType](ctx, taDst.api, nil)
	require.NoError(t, err)
	require.Len(t, list, 0)
}
