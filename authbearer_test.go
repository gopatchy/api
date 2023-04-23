package patchy_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gopatchy/patchy"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestBearerAuthSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	validToken := false

	ta.api.AddRequestHook(func(_ http.ResponseWriter, r *http.Request, _ *patchy.API) (*http.Request, error) {
		bearer := r.Context().Value(patchy.ContextAuthBearer)
		require.NotNil(t, bearer)
		require.IsType(t, &authBearerType{}, bearer)
		require.Equal(t, "foo", bearer.(*authBearerType).Name)
		require.Empty(t, bearer.(*authBearerType).Token)
		validToken = true
		return r, nil
	})

	ta.pyc.SetAuthToken("abcd")

	list, err := patchyc.List[authBearerType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)
	require.True(t, validToken)
}

func TestBearerAuthInvalidToken(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.AddRequestHook(func(_ http.ResponseWriter, r *http.Request, _ *patchy.API) (*http.Request, error) {
		require.Fail(t, "should not reach request hook")
		return r, nil
	})

	ta.pyc.SetAuthToken("bcde")

	_, err := patchyc.List[authBearerType](ctx, ta.pyc, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "token not found")
}

func TestBearerAuthOptional(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.List[authBearerType](ctx, ta.pyc, nil)
	require.NoError(t, err)
}
