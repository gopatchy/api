package patchy_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGo(t *testing.T) { //nolint:tparallel
	t.Parallel()

	paths, err := filepath.Glob("go_test/*_test.go.src")
	require.NoError(t, err)

	utils, err := filepath.Glob("go_test/*_util.go.src")
	require.NoError(t, err)

	utilsAbs := []string{}

	for _, util := range utils {
		abs, err := filepath.Abs(util)
		require.NoError(t, err)

		utilsAbs = append(utilsAbs, abs)
	}

	goRoot, err := os.MkdirTemp("", "go_root")
	require.NoError(t, err)

	goPath, err := os.MkdirTemp("", "go_path")
	require.NoError(t, err)

	cachePath, err := os.MkdirTemp("", "go_cache")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(goRoot)
		os.RemoveAll(goPath)
		os.RemoveAll(cachePath)
	})

	first := true

	for _, path := range paths {
		path, err := filepath.Abs(path)
		require.NoError(t, err)

		t.Run(
			filepath.Base(path),
			func(t *testing.T) {
				// This ugly hack makes sure we have cached dependencies up front
				if first {
					first = false
				} else {
					t.Parallel()
				}

				testGoPath(t, path, utilsAbs, goRoot, goPath, cachePath)
			},
		)
	}
}

func testGoPath(t *testing.T, src string, utils []string, goRoot, goPath, cachePath string) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("", "go_test")
	require.NoError(t, err)

	defer os.RemoveAll(dir)

	err = os.Symlink(src, filepath.Join(dir, strings.TrimSuffix(filepath.Base(src), ".src")))
	require.NoError(t, err)

	for _, util := range utils {
		err = os.Symlink(util, filepath.Join(dir, strings.TrimSuffix(filepath.Base(util), ".src")))
		require.NoError(t, err)
	}

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	tc, err := ta.pyc.GoClient(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tc)

	err = os.WriteFile(filepath.Join(dir, "client.go"), []byte(tc), 0o600)
	require.NoError(t, err)

	env := map[string]string{
		"BASE_URL": ta.baseBaseURL,
		"GOROOT":   goRoot,
		"GOPATH":   goPath,
		"GOCACHE":  cachePath,
	}

	gocmd := os.Getenv("GOCMD")
	if gocmd == "" {
		gocmd = "go"
	}

	runNoError(ctx, t, dir, env, gocmd, "mod", "init", "test")
	runNoError(ctx, t, dir, env, gocmd, "mod", "tidy")
	runNoError(ctx, t, dir, env, gocmd, "vet", ".")
	runNoError(ctx, t, dir, env, gocmd, "test", ".")

	ta.checkTests(t)
}
