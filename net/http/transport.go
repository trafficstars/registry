package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

const DefaultMaxRetry = 4

type Transport struct {
	MaxRetry int
	http.Transport
}

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
		if backend, err = _balancer.nextRoundRobin(service); err == nil {
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
