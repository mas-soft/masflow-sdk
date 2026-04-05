package masflowsdk

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

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
