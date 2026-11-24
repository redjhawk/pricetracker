package httpapi

import (
	"net/http"

	"pricefollower.local/config"
)

// NewHTTPServer returns the application's HTTP server (issue #273). Not implemented yet.
func NewHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{}
}
