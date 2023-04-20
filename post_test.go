package api_test

import (
	"context"
	"testing"

	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestPOST(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "foo", created.Text)
	require.NotEmpty(t, created.ID)

	get, err := patchyc.Get[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", get.Text)
	require.Equal(t, created.ID, get.ID)
}
