package http

import "net/http"

type Transport struct {
	http.Transport
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		err      error
		backend  *backend
		response *http.Response
		service  = req.URL.Host
	)

	for i := 0; i <= _balancer.countOfBackends(service)*2; i++ {
		if backend, err = _balancer.next(service); err == nil {
			req.URL.Host = backend.address
			if response, err = t.Transport.RoundTrip(req); err == nil {
				return response, nil
			}
			backend.skip()
		}
	}
	return nil, err
}
