package httpapi

import (
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"pricefollower.local/config"
)

// Limits of the application's HTTP server (issue #273).
const (
	serverReadHeaderTimeout = 5 * time.Second
	serverReadTimeout       = 15 * time.Second // whole request: headers and body
	serverWriteTimeout      = 60 * time.Second // from the received headers; Settings save waits up to 20 s for Claude
	serverIdleTimeout       = 60 * time.Second
	serverMaxHeaderBytes    = 64 << 10
)

// NewHTTPServer returns the application's HTTP server: handler on the configured host and port with the
// request limits above. One server serves development and production mode.
func NewHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}
}

// abortLateBodies ends the request without a response when its body is not received within the server's
// read limit (FR-HTTP-TMO-001); the handler would otherwise answer 400 INVALID_JSON.
func abortLateBodies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = lateBody{r.Body}
		}
		next.ServeHTTP(w, r)
	})
}

type lateBody struct{ io.ReadCloser }

func (b lateBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		panic(http.ErrAbortHandler) // net/http closes the connection, writes nothing and logs nothing
	}
	return n, err
}
