package patchy_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func runNoError(ctx1 context.Context, t *testing.T, dir string, env map[string]string, name string, arg ...string) {
	ctx2, cancel := context.WithCancel(ctx1)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx2, name, arg...)

	if dir != "" {
		cmd.Dir = dir
	}

	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	out, err := cmd.Output()
	t.Logf("dir='%s'\ncmd='%s'\nargs=%v\nout='%s'\nerr='%s'", dir, name, arg, string(out), getStderr(err))

	if err != nil && strings.Contains(err.Error(), "signal: killed") {
		return
	}

	require.NoError(t, err)
}

func getStderr(err error) string {
	ee := &exec.ExitError{}
	if errors.As(err, &ee) {
		return string(ee.Stderr)
	}

	return ""
}
