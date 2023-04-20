package api_test

import (
	"context"
	"testing"

	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo", Num: 1})
	require.NoError(t, err)

	updated, err := patchyc.Update[testType](ctx, ta.pyc, created.ID, &testTypeRequest{Text: patchyc.P("bar")}, nil)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "bar", updated.Text)
	require.EqualValues(t, 1, updated.Num)
	require.EqualValues(t, created.Generation+1, updated.Generation)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
	require.EqualValues(t, 1, get.Num)
	require.EqualValues(t, created.Generation+1, updated.Generation)
}

func TestUpdateNotExist(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	updated, err := patchyc.Update[testType](ctx, ta.pyc, "doesnotexist", &testType{Text: "bar"}, nil)
	require.Error(t, err)
	require.Nil(t, updated)
}

func TestUpdateIfMatchETagSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	updated, err := patchyc.Update[testType](ctx, ta.pyc, created.ID, &testType{Text: "bar"}, &patchyc.UpdateOpts{Prev: created})
	require.NoError(t, err)
	require.Equal(t, "bar", updated.Text)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "bar", get.Text)
}

func TestUpdateIfMatchETagMismatch(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	created.ETag = "etag:doesnotmatch"

	updated, err := patchyc.Update[testType](ctx, ta.pyc, created.ID, &testType{Text: "bar"}, &patchyc.UpdateOpts{Prev: created})
	require.Error(t, err)
	require.Nil(t, updated)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
}

func TestUpdateIfMatchInvalid(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	ta.pyc.SetHeader("If-Match", `"foobar"`)

	updated, err := patchyc.Update[testType](ctx, ta.pyc, created.ID, &testType{Text: "bar"}, nil)
	require.Error(t, err)
	require.Nil(t, updated)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
}
