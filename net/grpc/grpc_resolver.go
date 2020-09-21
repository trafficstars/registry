package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/resolver"

	net_balancer "github.com/trafficstars/registry/net/balancer"
)

const defaultRefreshInterval = time.Second * 5

type grpcMetadata struct {
	serviceName          string
	servicePort          string
	backend              *net_balancer.Backend
	balancer             net_balancer.Balancer
	maxRequestsByBackend int
}

type grpcResolver struct {
	// Service name in the discovery registry
	serviceName string

	// Service port number
	servicePort string

	// Maximal amount of requests by backend
	maxRequestsByBackend int

	// Default connection balancer
	balancer net_balancer.Balancer

	// Refresh timer interval
	freq time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	cc     resolver.ClientConn

	t *time.Ticker
}

// ResolveNow invoke an immediate resolution of the target that this dnsResolver watches.
func (r *grpcResolver) ResolveNow(opt resolver.ResolveNowOption) {}

// Close closes the dnsResolver.
func (r *grpcResolver) Close() {
	r.cancel()
	if r.t != nil {
		r.t.Stop()
	}
}

func (r *grpcResolver) watcher() {
	r.t = time.NewTicker(r.freq)
	for {
		select {
		case <-r.t.C:
		case <-r.ctx.Done():
			return
		}
		r.refreshConnection()
	}
}

func (r *grpcResolver) refreshConnection() {
	var (
		service     = r.serviceName
		balancer    = r.balancer
		addressList []resolver.Address
	)
	if balancer == nil {
		balancer = net_balancer.Default()
	}

	backends := balancer.Backends(service)
	for _, backend := range backends {
		address := backend.Address()
		if r.servicePort != "" {
			address = backend.Hostname() + ":" + r.servicePort
		}
		addressList = append(addressList, resolver.Address{
			Addr: address,
			Metadata: &grpcMetadata{
				serviceName:          service,
				servicePort:          r.servicePort,
				backend:              backend,
				balancer:             balancer,
				maxRequestsByBackend: r.maxRequestsByBackend,
			},
		})
	}

	r.cc.NewAddress(addressList)
}

var _ resolver.Resolver = (*grpcResolver)(nil)
