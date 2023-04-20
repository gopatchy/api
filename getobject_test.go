package patchy_test

import (
	"context"
	"testing"

	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestGetPrev(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.EqualValues(t, 0, get.Num)

	// Validate that previous version passing only compares the ETag
	get.Num = 1

	get2, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, &patchyc.GetOpts{Prev: get})
	require.NoError(t, err)
	require.Equal(t, "foo", get2.Text)
	require.EqualValues(t, 1, get2.Num)
}
