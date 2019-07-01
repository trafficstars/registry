package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

const DefaultMaxRetry = 2

// Transport wrapper of the HTTP transport object
type Transport struct {
	MaxRetry             int
	MaxRequestsByBackend int
	http.Transport
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		err      error
		body     []byte
		backend  *backend
		response *http.Response
		service  = req.URL.Host
		maxRetry = t.MaxRetry
	)
	if maxRetry == 0 {
		maxRetry = DefaultMaxRetry
	}
	if req.Body != nil {
		body, _ = ioutil.ReadAll(req.Body)
	}
	for i := 0; i <= maxRetry; i++ {
		if backend, err = _balancer.next(service, t.MaxRequestsByBackend); err == nil {
			backend.incConcurrentRequest(1)
			defer backend.incConcurrentRequest(-1)

			req.URL.Host = backend.address
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			if response, err = t.Transport.RoundTrip(req); err == nil {
				return response, nil
			}
			backend.skip()
		}
	}
	return nil, err
}

var _ http.RoundTripper = (*Transport)(nil)
