package gotest

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"test/goclient"
)

func registerTest(t *testing.T) func() {
	logEvent(t, "begin")

	return func() {
		logEvent(t, "end")
	}
}

func getClient(t *testing.T) *goclient.Client {
	return goclient.NewClient(getBaseURL(t)).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
}

func getResty(t *testing.T) *resty.Request {
	return resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetBaseURL(getBaseURL(t)).
		R()
}

func logEvent(t *testing.T, event string) {
	getResty(t).
		SetQueryParam("event", event).
		SetQueryParam("name", t.Name()).
		Get("api/_logEvent")
	// Allowed to fail
}

func closeAllConns(t *testing.T) {
	logEvent(t, "connsClose")
	// Our own conn gets killed; give it time for others to die as well
	time.Sleep(100 * time.Millisecond)
}

func getBaseURL(t *testing.T) string {
	baseURL := os.Getenv("BASE_URL")
	require.NotEmpty(t, baseURL)

	return baseURL
}
