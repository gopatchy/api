package patchy_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
)

func TestTSNode(t *testing.T) {
	t.Parallel()

	testTS(t, "node", testPathNode)
}

func TestTSFirefox(t *testing.T) {
	t.Parallel()

	testTS(t, "browser", testPathBrowser(runFirefox))
}

func TestTSChrome(t *testing.T) {
	t.Parallel()

	testTS(t, "browser", testPathBrowser(runChrome))
}

func testTS(t *testing.T, env string, runner func(*testing.T, *testAPI, string)) {
	dir := buildTS(t, env)
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	paths, err := filepath.Glob(filepath.Join(dir, "*_test.js"))
	require.NoError(t, err)

	for _, path := range paths {
		path := path

		t.Run(
			filepath.Base(path),
			func(t *testing.T) {
				t.Parallel()

				ta := newTestAPIInsecure(t)
				defer ta.shutdown(t)

				runner(t, ta, path)

				ta.checkTests(t)
			},
		)
	}
}

func testPathNode(t *testing.T, ta *testAPI, path string) {
	env := map[string]string{
		"NODE_DEBUG":                   os.Getenv("NODE_DEBUG"),
		"NODE_NO_WARNINGS":             "1",
		"NODE_TLS_REJECT_UNAUTHORIZED": "0",
		"BASE_URL":                     ta.baseURL,
	}

	ctx := context.Background()

	runNoError(ctx, t, filepath.Dir(path), env, "node", "--enable-source-maps", filepath.Base(path))
}

func testPathBrowser(runCmd func(context.Context, *testing.T, string, string)) func(*testing.T, *testAPI, string) {
	return func(t *testing.T, ta *testAPI, path string) {
		ctx, cancel := context.WithCancel(context.Background())

		ss, ssBase := newStaticServer(t, filepath.Dir(path), ta.baseURL)
		defer func() {
			err := ss.Shutdown(ctx)
			require.NoError(t, err)
		}()

		go func() {
			<-ta.testDone
			cancel()
		}()

		profileDir, err := os.MkdirTemp("", "browser_profile")
		require.NoError(t, err)

		defer os.RemoveAll(profileDir)

		url := fmt.Sprintf("%shtml/%s", ssBase, strings.TrimSuffix(filepath.Base(path), ".js"))
		t.Logf("URL: %s", url)

		runCmd(ctx, t, profileDir, url)
	}
}

func runFirefox(ctx context.Context, t *testing.T, profileDir, url string) {
	runNoError(ctx, t, "", nil, "firefox", "--headless", "--no-remote", "--profile", profileDir, url)
}

func runChrome(ctx context.Context, t *testing.T, profileDir, url string) {
	runNoError(ctx, t, "", nil, "google-chrome", "--headless", "--disable-gpu", "--remote-debugging-port=9222", fmt.Sprintf("--user-data-dir=%s", profileDir), url)
}

func buildTS(t *testing.T, env string) string {
	dir, err := os.MkdirTemp("", "ts_test")
	require.NoError(t, err)

	paths, err := filepath.Glob("ts_test/*")
	require.NoError(t, err)

	for _, path := range paths {
		src, err := filepath.Abs(path)
		require.NoError(t, err)

		base := filepath.Base(path)

		if strings.Contains(base, "__") {
			parts := strings.SplitN(base, "__", 2)

			if parts[0] == env {
				base = parts[1]
			} else {
				continue
			}
		}

		err = os.Symlink(src, filepath.Join(dir, base))
		require.NoError(t, err)
	}

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	tc, err := ta.pyc.TSClient(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tc)

	err = os.WriteFile(filepath.Join(dir, "client.ts"), []byte(tc), 0o600)
	require.NoError(t, err)

	runNoError(ctx, t, dir, nil, "tsc", "--pretty")

	return dir
}

func newStaticServer(t *testing.T, dir, baseURL string) (*http.Server, string) {
	r := httprouter.New()

	r.ServeFiles("/js/*filepath", http.Dir(dir))

	r.Handle("GET", "/html/*filepath", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")

		fmt.Fprintf(w, `<!DOCTYPE html>
<link rel="icon" href="data:,">

<script>
globalThis.baseURL = "%s";
</script>

<script type="module" src="../js%s.js"></script>
`, baseURL, params.ByName("filepath"))
	})

	srv := &http.Server{
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}

	l, err := net.Listen("tcp", "[::]:0")
	require.NoError(t, err)

	go func() {
		err := srv.Serve(l)
		require.ErrorIs(t, err, http.ErrServerClosed)
	}()

	return srv, fmt.Sprintf("http://[::1]:%d/", l.Addr().(*net.TCPAddr).Port)
}
