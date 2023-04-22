package patchy_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gopatchy/patchy"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestBasicAuthSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	validUser := false

	ta.api.SetRequestHook(func(r *http.Request, _ *patchy.API) (*http.Request, error) {
		basic := r.Context().Value(patchy.ContextAuthBasic)
		require.NotNil(t, basic)
		require.IsType(t, &authBasicType{}, basic)
		require.Equal(t, "foo", basic.(*authBasicType).User)
		require.Empty(t, basic.(*authBasicType).Pass)
		validUser = true
		return r, nil
	})

	ta.pyc.SetBasicAuth("foo", "abcd")

	list, err := patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)
	require.True(t, validUser)
}

func TestBasicAuthInvalidUser(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(func(r *http.Request, _ *patchy.API) (*http.Request, error) {
		require.Fail(t, "should not reach request hook")
		return r, nil
	})

	ta.pyc.SetBasicAuth("bar", "abcd")

	_, err := patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "user not found")
}

func TestBasicAuthInvalidPassword(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(func(r *http.Request, _ *patchy.API) (*http.Request, error) {
		require.Fail(t, "should not reach request hook")
		return r, nil
	})

	ta.pyc.SetBasicAuth("foo", "bcde")

	_, err := patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "user password mismatch")
}

func TestBasicAuthOptional(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.NoError(t, err)
}
