package patchy_test

import (
	"context"
	"testing"

	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestDeleteStream(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchyc.StreamGet[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.NoError(t, stream.Error())
	require.Equal(t, "foo", s1.Text)

	err = patchyc.Delete[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	s2 := stream.Read()
	require.Nil(t, s2, stream.Error())
	require.Error(t, stream.Error())
}

func TestDeleteIfMatchETagSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	err = patchyc.Delete[testType](ctx, ta.pyc, created.ID, &patchyc.UpdateOpts{Prev: created})
	require.NoError(t, err)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Nil(t, get)
}

func TestDeleteIfMatchETagMismatch(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Update[testType](ctx, ta.pyc, created.ID, &testType{Text: "bar"}, nil)
	require.NoError(t, err)

	err = patchyc.Delete[testType](ctx, ta.pyc, created.ID, &patchyc.UpdateOpts{Prev: created})
	require.Error(t, err)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, get)
	require.Equal(t, "bar", get.Text)
}
