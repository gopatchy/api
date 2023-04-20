package patchy_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gopatchy/patchy"
	"github.com/stretchr/testify/require"
)

type complexTestType struct {
	patchy.Metadata
	A string
	B int
	C []string
	D nestedType
	E *nestedType
}

type nestedType struct {
	F []int
	G string
}

func TestTemplateGoClient(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	patchy.Register[complexTestType](ta.api)

	ctx := context.Background()

	gc, err := ta.pyc.GoClient(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gc)

	t.Log(gc)

	dir, err := os.MkdirTemp("", "goclient")
	require.NoError(t, err)

	defer os.RemoveAll(dir)

	err = os.WriteFile(filepath.Join(dir, "client.go"), []byte(gc), 0o600)
	require.NoError(t, err)

	runNoError(ctx, t, dir, nil, "go", "mod", "init", "test")
	runNoError(ctx, t, dir, nil, "go", "mod", "tidy")
	runNoError(ctx, t, dir, nil, "go", "vet", ".")
	runNoError(ctx, t, dir, nil, "go", "build", ".")
}
