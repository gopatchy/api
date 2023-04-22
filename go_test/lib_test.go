package gotest

import (
	"crypto/tls"
	"os"
	"testing"

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

func logEvent(t *testing.T, event string) {
	resp, err := resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetBaseURL(getBaseURL(t)).
		R().
		SetQueryParam("event", event).
		SetQueryParam("name", t.Name()).
		Get("api/_logEvent")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess())
}

func getBaseURL(t *testing.T) string {
	baseURL := os.Getenv("BASE_URL")
	require.NotEmpty(t, baseURL)

	return baseURL
}