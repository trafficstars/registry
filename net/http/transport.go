package http

import (
	"bytes"
	"io/ioutil"
	"net/http"

	regbalancer "github.com/trafficstars/registry/net/balancer"
)

// Option type
type Option func(opt *Transport)

// Balancer option setup
func Balancer(balancer regbalancer.Balancer) Option {
	return func(opt *Transport) {
		opt.balancer = balancer
	}
}

// MaxRequestsByBackend option setup
func MaxRequestsByBackend(maxRequestsByBackend int) Option {
	return func(opt *Transport) {
		opt.maxRequestsByBackend = maxRequestsByBackend
	}
}

// MaxRetry option setup
func MaxRetry(maxRetry int) Option {
	return func(opt *Transport) {
		opt.maxRetry = maxRetry
	}
}

// DefaultMaxRetry count
const DefaultMaxRetry = 2

// Transport wrapper of the HTTP transport object
type Transport struct {
	// max retry attempts before fail
	maxRetry int

	// Max concurrent requests by backend
	maxRequestsByBackend int

	// Balancer default for this RoundTripper
	balancer regbalancer.Balancer

	// Target HTTP transport
	httpTransport *http.Transport
}

// WrapHTTPTransport by original transport object
func WrapHTTPTransport(transport *http.Transport, options ...Option) *Transport {
	wrapper := &Transport{httpTransport: transport}
	for _, opt := range options {
		opt(wrapper)
	}
	if wrapper.maxRetry <= 0 {
		wrapper.maxRetry = DefaultMaxRetry
	}
	if wrapper.balancer == nil {
		wrapper.balancer = regbalancer.Default()
	}
	return wrapper
}

// HTTPTransport returns wrapped transport object
func (t *Transport) HTTPTransport() *http.Transport {
	return t.httpTransport
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		err      error
		body     []byte
		backend  *regbalancer.Backend
		response *http.Response
		service  = req.URL.Host
	)
	if req.Body != nil {
		body, _ = ioutil.ReadAll(req.Body)
	}
	for i := 0; i <= t.maxRetry; i++ {
		if backend, err = t.balancer.Next(service, t.maxRequestsByBackend); err == nil {
			// Mark backend as performing a request
			backend.IncConcurrentRequest(1)
			defer backend.IncConcurrentRequest(-1)

			req.URL.Host = backend.Address()
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			if response, err = t.httpTransport.RoundTrip(req); err == nil {
				return response, nil
			}

			// Skip next tries of requests to this backend
			backend.Skip()
		}
	}
	return nil, err
}

var _ http.RoundTripper = (*Transport)(nil)
