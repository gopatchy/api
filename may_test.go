//nolint:goerr113
package patchy_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gopatchy/patchy"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

type mayType struct {
	patchy.Metadata
	Text1 string
}

func (mt *mayType) MayRead(ctx context.Context, api *patchy.API) error {
	if ctx.Value(refuseRead) != nil {
		return fmt.Errorf("may not read")
	}

	t1r := ctx.Value(text1Read)
	if t1r != nil {
		mt.Text1 = t1r.(string)
	}

	nt1 := ctx.Value(newText1)
	if nt1 != nil {
		// Use a separate context so we don't recursively create objects
		_, err := patchy.Create[mayType](context.Background(), api, &mayType{Text1: nt1.(string)}) //nolint:contextcheck
		if err != nil {
			return err
		}
	}

	return nil
}

func (mt *mayType) MayWrite(ctx context.Context, prev *mayType, _ *patchy.API) error {
	if ctx.Value(refuseWrite) != nil {
		return fmt.Errorf("may not write")
	}

	t1w := ctx.Value(text1Write)
	if t1w != nil {
		mt.Text1 = t1w.(string)
	}

	return nil
}

type contextKey int

const (
	refuseRead contextKey = iota
	refuseWrite
	text1Read
	text1Write
	newText1
)

func requestHook(r *http.Request, _ *patchy.API) (*http.Request, error) {
	ctx := r.Context()

	if r.Header.Get("X-Refuse-Read") != "" {
		ctx = context.WithValue(ctx, refuseRead, true)
	}

	if r.Header.Get("X-Refuse-Write") != "" {
		ctx = context.WithValue(ctx, refuseWrite, true)
	}

	t1r := r.Header.Get("X-Text1-Read")
	if t1r != "" {
		ctx = context.WithValue(ctx, text1Read, t1r)
	}

	t1w := r.Header.Get("X-Text1-Write")
	if t1w != "" {
		ctx = context.WithValue(ctx, text1Write, t1w)
	}

	nt1 := r.Header.Get("X-NewText1")
	if nt1 != "" {
		ctx = context.WithValue(ctx, newText1, nt1)
	}

	return r.WithContext(ctx), nil
}

func TestMayWriteCreateSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)
}

func TestMayWriteCreateRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	ta.pyc.SetHeader("X-Refuse-Write", "x")

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.Error(t, err)
	require.Nil(t, created)
}

func TestMayWriteReplaceSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	_, err = patchyc.Replace[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.NoError(t, err)
}

func TestMayWriteReplaceRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Write", "x")

	_, err = patchyc.Replace[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.Error(t, err)
}

func TestMayWriteUpdateSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	_, err = patchyc.Update[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.NoError(t, err)
}

func TestMayWriteUpdateRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Write", "x")

	_, err = patchyc.Update[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.Error(t, err)
}

func TestMayWriteDeleteSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	err = patchyc.Delete[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
}

func TestMayWriteDeleteRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Write", "x")

	err = patchyc.Delete[mayType](ctx, ta.pyc, created.ID, nil)
	require.Error(t, err)
}

func TestMayReadGetSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, get)
}

func TestMayReadGetRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.Error(t, err)
	require.Nil(t, get)
}

func TestMayReadStreamGetSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	stream, err := patchyc.StreamGet[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	ev := stream.Read()
	require.NotNil(t, ev, stream.Error())
}

func TestMayReadStreamGetRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	stream, err := patchyc.StreamGet[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	ev := stream.Read()
	require.Nil(t, ev)
}

func TestMayReadListSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	list, err := patchyc.List[mayType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestMayReadListRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	list, err := patchyc.List[mayType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestMayReadStreamListSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	stream, err := patchyc.StreamList[mayType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 1)
}

func TestMayReadStreamListRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	stream, err := patchyc.StreamList[mayType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.NotNil(t, stream)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Len(t, s1, 0)
}

func TestMayReadCreateSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)
}

func TestMayReadCreateRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.Error(t, err)
}

func TestMayReadReplaceSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	_, err = patchyc.Replace[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.NoError(t, err)
}

func TestMayReadReplaceRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	_, err = patchyc.Replace[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.Error(t, err)
}

func TestMayReadUpdateSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	_, err = patchyc.Update[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.NoError(t, err)
}

func TestMayReadUpdateRefuse(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Refuse-Read", "x")

	_, err = patchyc.Update[mayType](ctx, ta.pyc, created.ID, &mayType{}, nil)
	require.Error(t, err)
}

func TestMayWriteMutateCreate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	ta.pyc.SetHeader("X-Text1-Write", "1234")

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "1234", get.Text1)
}

func TestMayWriteMutateReplace(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Write", "2345")

	_, err = patchyc.Replace[mayType](ctx, ta.pyc, created.ID, &mayType{Text1: "bar"}, nil)
	require.NoError(t, err)

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "2345", get.Text1)
}

func TestMayWriteMutateUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Write", "3456")

	_, err = patchyc.Update[mayType](ctx, ta.pyc, created.ID, &mayType{Text1: "bar"}, nil)
	require.NoError(t, err)

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "3456", get.Text1)
}

func TestMayReadMutateGet(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Read", "1234")

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "1234", get.Text1)
}

func TestMayReadMutateCreate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	ta.pyc.SetHeader("X-Text1-Read", "2345")

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)
	require.Equal(t, "2345", created.Text1)

	ta.pyc.SetHeader("X-Text1-Read", "")

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text1)
}

func TestMayReadMutateReplace(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Read", "3456")

	replaced, err := patchyc.Replace[mayType](ctx, ta.pyc, created.ID, &mayType{Text1: "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, "3456", replaced.Text1)

	ta.pyc.SetHeader("X-Text1-Read", "")

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text1)
}

func TestMayReadMutateUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Read", "4567")

	updated, err := patchyc.Update[mayType](ctx, ta.pyc, created.ID, &mayType{Text1: "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, "4567", updated.Text1)

	ta.pyc.SetHeader("X-Text1-Read", "")

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text1)
}

func TestMayReadMutateStreamGet(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Read", "5678")

	stream, err := patchyc.StreamGet[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1)
	require.Equal(t, "5678", s1.Text1)
}

func TestMayReadMutateList(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Read", "6789")

	list, err := patchyc.List[mayType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "6789", list[0].Text1)
}

func TestMayReadMutateStreamList(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	_, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-Text1-Read", "789a")

	stream, err := patchyc.StreamList[mayType](ctx, ta.pyc, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "789a", s1[0].Text1)
}

func TestMayReadSideEffect(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(requestHook)
	patchy.Register[mayType](ta.api)

	created, err := patchyc.Create[mayType](ctx, ta.pyc, &mayType{Text1: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("X-NewText1", "abcd")

	get, err := patchyc.Get[mayType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text1)

	ta.pyc.SetHeader("X-NewText1", "")

	list, err := patchyc.List[mayType](ctx, ta.pyc, &patchyc.ListOpts{Sorts: []string{"+text1"}})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "abcd", list[0].Text1)
	require.Equal(t, "foo", list[1].Text1)
}
