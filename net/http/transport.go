package http

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/trafficstars/registry/net/balancer"
)

// DefaultMaxRetry count
const DefaultMaxRetry = 2

// Transport wrapper of the HTTP transport object
type Transport struct {
	// MaxRetry attempts before fail
	MaxRetry int

	// Max concurrent requests by backend
	MaxRequestsByBackend int

	// Balancer default for this RoundTripper
	Balancer balancer.Balancer

	// Target HTTP transport
	http.Transport
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		err      error
		body     []byte
		backend  *balancer.Backend
		response *http.Response
		service  = req.URL.Host
		maxRetry = t.MaxRetry
		balancer = t.Balancer
	)
	if maxRetry == 0 {
		maxRetry = DefaultMaxRetry
	}
	if balancer == nil {
		balancer = balancer.Default()
	}
	if req.Body != nil {
		body, _ = ioutil.ReadAll(req.Body)
	}
	for i := 0; i <= maxRetry; i++ {
		if backend, err = balancer.Next(service, t.MaxRequestsByBackend); err == nil {
			backend.IncConcurrentRequest(1)
			defer backend.IncConcurrentRequest(-1)

			req.URL.Host = backend.Address()
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			if response, err = t.Transport.RoundTrip(req); err == nil {
				return response, nil
			}

			// Skip next tries of requests to this backend
			backend.Skip()
		}
	}
	return nil, err
}

var _ http.RoundTripper = (*Transport)(nil)
