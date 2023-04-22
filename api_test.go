package patchy_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/gopatchy/patchy"
	"github.com/stretchr/testify/require"
)

func TestRegisterMissingMetadata(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.Panics(t, func() {
		patchy.Register[missingMetadata](api)
	})
}

func TestIsSafeSuccess(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	patchy.Register[testType3](api)

	require.NoError(t, api.IsSafe())
}

func TestIsSafeWithoutWrite(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NoError(t, api.IsSafe())

	patchy.Register[testType](api)

	require.ErrorIs(t, api.IsSafe(), patchy.ErrMissingAuthCheck)
}

func TestIsSafeWithoutRead(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NoError(t, api.IsSafe())

	patchy.Register[testType2](api)

	require.ErrorIs(t, api.IsSafe(), patchy.ErrMissingAuthCheck)
}

func TestCheckSafeSuccess(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	patchy.Register[testType3](api)

	require.NotPanics(t, api.CheckSafe)
}

func TestCheckSafeWithoutWrite(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NotPanics(t, api.CheckSafe)

	patchy.Register[testType](api)

	require.Panics(t, api.CheckSafe)
}

func TestCheckSafeWithoutRead(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NotPanics(t, api.CheckSafe)

	patchy.Register[testType2](api)

	require.Panics(t, api.CheckSafe)
}

func TestAcceptJSON(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	get := &testType{}

	resp, err := ta.r().
		SetHeader("Accept", "text/xml, application/json").
		SetResult(get).
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.False(t, resp.IsError())
	require.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	require.Equal(t, "foo", get.Text)
	require.Equal(t, created.ID, get.ID)
}

func TestAcceptEventStream(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	resp, err := ta.r().
		SetDoNotParseResponse(true).
		SetHeader("Accept", "text/event-stream, text/xml").
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.False(t, resp.IsError())
	require.Equal(t, "text/event-stream", resp.Header().Get("Content-Type"))
	resp.RawBody().Close()
}

func TestAcceptFailure(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	resp, err := ta.r().
		SetHeader("Accept", "unsupported").
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.True(t, resp.IsError())
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode())
}

func TestAcceptListFailure(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	resp, err := ta.r().
		SetHeader("Accept", "unsupported").
		Get("testtype")
	require.NoError(t, err)
	require.True(t, resp.IsError())
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode())
}

func TestRequestHookError(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchy.Create[testType](ctx, ta.api, &testType{Text: "foo"})
	require.NoError(t, err)

	ta.api.SetRequestHook(func(*http.Request, *patchy.API) (*http.Request, error) {
		return nil, fmt.Errorf("test reject") //nolint:goerr113
	})

	get := &testType{}

	resp, err := ta.r().
		SetResult(get).
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.True(t, resp.IsError())
	require.Contains(t, resp.String(), "test reject")
}
