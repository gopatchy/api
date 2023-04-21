package patchy_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGo(t *testing.T) {
	t.Parallel()

	dir, testDir, env, tests := buildGo(t)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	for _, test := range tests {
		test := test

		t.Run(
			test,
			func(t *testing.T) {
				t.Parallel()
				testGoPath(t, testDir, env, test)
			},
		)
	}
}

func testGoPath(t *testing.T, testDir string, env map[string]string, test string) {
	ctx := context.Background()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	env2 := map[string]string{}
	for k, v := range env {
		env2[k] = v
	}

	env2["BASE_URL"] = ta.baseBaseURL

	runNoError(ctx, t, testDir, env2, goCmd(), "test", "-run", fmt.Sprintf("^%s$", test))

	ta.checkTests(t)
}

func buildGo(t *testing.T) (string, string, map[string]string, []string) {
	dir, err := os.MkdirTemp("", "go_test")
	require.NoError(t, err)

	goRootDir := filepath.Join(dir, "root")
	goPathDir := filepath.Join(dir, "path")
	goCacheDir := filepath.Join(dir, "cache")
	workDir := filepath.Join(dir, "work")
	goClientDir := filepath.Join(dir, "work/goclient")
	testDir := filepath.Join(dir, "work/gotest")

	require.NoError(t, os.MkdirAll(goRootDir, 0o700))
	require.NoError(t, os.MkdirAll(goPathDir, 0o700))
	require.NoError(t, os.MkdirAll(goCacheDir, 0o700))
	require.NoError(t, os.MkdirAll(goClientDir, 0o700))
	require.NoError(t, os.MkdirAll(testDir, 0o700))

	paths, err := filepath.Glob("go_test/*")
	require.NoError(t, err)

	for _, path := range paths {
		src, err := filepath.Abs(path)
		require.NoError(t, err)

		base := strings.TrimSuffix(filepath.Base(path), ".src")

		err = os.Symlink(src, filepath.Join(testDir, base))
		require.NoError(t, err)
	}

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	gc, err := ta.pyc.GoClient(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gc)

	err = os.WriteFile(filepath.Join(dir, "work/goclient/client.go"), []byte(gc), 0o600)
	require.NoError(t, err)

	env := map[string]string{
		"GOROOT":  goRootDir,
		"GOPATH":  goPathDir,
		"GOCACHE": goCacheDir,
	}

	runNoError(ctx, t, goClientDir, env, goCmd(), "mod", "init", "test/goclient")
	runNoError(ctx, t, goClientDir, env, goCmd(), "mod", "tidy")

	runNoError(ctx, t, testDir, env, goCmd(), "mod", "init", "test/gotest")
	runNoError(ctx, t, testDir, env, goCmd(), "mod", "tidy")

	err = os.WriteFile(filepath.Join(workDir, "go.work"), []byte(`
go 1.20

use (
	./goclient
	./gotest
)
`), 0o600)
	require.NoError(t, err)

	runNoError(ctx, t, goClientDir, env, goCmd(), "vet")
	runNoError(ctx, t, testDir, env, goCmd(), "vet")

	testBlob := runNoError(ctx, t, testDir, env, goCmd(), "test", "-list", ".")

	tests := []string{}

	for _, line := range strings.Split(testBlob, "\n") {
		if strings.HasPrefix(line, "Test") {
			tests = append(tests, line)
		}
	}

	return dir, testDir, env, tests
}

func goCmd() string {
	gocmd := os.Getenv("GOCMD")
	if gocmd != "" {
		return gocmd
	}

	return "go"
}
