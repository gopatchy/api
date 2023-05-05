package patchy

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
)

type DebugInfo struct {
	Server *ServerInfo `json:"server"`
	IP     *IPInfo     `json:"ip"`
	HTTP   *HTTPInfo   `json:"http"`
	TLS    *TLSInfo    `json:"tls"`
}

type ServerInfo struct {
	Hostname string `json:"hostname"`
}

type IPInfo struct {
	RemoteAddr string `json:"remoteAddr"`
}

type HTTPInfo struct {
	Protocol string      `json:"protocol"`
	Method   string      `json:"method"`
	Header   http.Header `json:"header"`
	URL      string      `json:"url"`
}

type TLSInfo struct {
	Version            uint16 `json:"version"`
	DidResume          bool   `json:"didResume"`
	CipherSuite        uint16 `json:"cipherSuite"`
	NegotiatedProtocol string `json:"negotiatedProtocol"`
	ServerName         string `json:"serverName"`
}

func (api *API) handleDebug(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	api.AddEventData(ctx, "operation", "debug")

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")

	if r.TLS == nil {
		r.TLS = &tls.ConnectionState{}
	}

	_ = enc.Encode(&DebugInfo{ //nolint:errchkjson
		Server: buildServerInfo(),
		IP:     buildIPInfo(r),
		HTTP:   buildHTTPInfo(r),
		TLS:    buildTLSInfo(r),
	})
}

func buildServerInfo() *ServerInfo {
	hostname, _ := os.Hostname()

	return &ServerInfo{
		Hostname: hostname,
	}
}

func buildIPInfo(r *http.Request) *IPInfo {
	return &IPInfo{
		RemoteAddr: r.RemoteAddr,
	}
}

func buildHTTPInfo(r *http.Request) *HTTPInfo {
	return &HTTPInfo{
		Protocol: r.Proto,
		Method:   r.Method,
		Header:   r.Header,
		URL:      r.URL.String(),
	}
}

func buildTLSInfo(r *http.Request) *TLSInfo {
	return &TLSInfo{
		Version:            r.TLS.Version,
		DidResume:          r.TLS.DidResume,
		CipherSuite:        r.TLS.CipherSuite,
		NegotiatedProtocol: r.TLS.NegotiatedProtocol,
		ServerName:         r.TLS.ServerName,
	}
}
