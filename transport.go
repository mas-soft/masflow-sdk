package masflowsdk

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/http2"
)

type protocolMode uint8

const (
	protocolAuto protocolMode = iota
	protocolConnect
	protocolGRPC
)

func shouldUseGRPC(baseURL string, mode protocolMode) bool {
	switch mode {
	case protocolConnect:
		return false
	case protocolGRPC:
		return true
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return true
	}

	return parsed.Scheme != "http"
}

func usesPlaintextHTTP(baseURL string) bool {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	return parsed.Scheme == "http"
}

// newH2CClient returns an [http.Client] that speaks HTTP/2 over cleartext (h2c).
// This is required for gRPC over plaintext (http://) connections because Go's
// default HTTP client only negotiates HTTP/2 when TLS is present.
//
// For TLS (https://) connections the standard [http.DefaultClient] already
// supports HTTP/2 via ALPN, so this client is only needed for plaintext.
func newH2CClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				// For h2c we dial a plain TCP connection, ignoring the TLS config.
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
}
