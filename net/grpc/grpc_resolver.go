package grpc

import (
	"context"
	"time"
	"log"

	"google.golang.org/grpc/resolver"

	net_balancer "github.com/trafficstars/registry/net/balancer"
)

const defaultRefreshInterval = time.Second * 5

type grpcMetadata struct {
	serviceName          string
	backend              *net_balancer.Backend
	balancer             net_balancer.Balancer
	maxRequestsByBackend int
}

type grpcResolver struct {
	// Service name in the discovery registry
	serviceName string

	// Maximal amount of requests by backend
	maxRequestsByBackend int

	// Backend used at the moment
	currentBackend *net_balancer.Backend

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
func (r *grpcResolver) ResolveNow(opt resolver.ResolveNowOption) {
	r.refreshConnection()
}

// Close closes the dnsResolver.
func (r *grpcResolver) Close() {
	r.cancel()
	if r.t != nil {
		r.t.Stop()
	}
}

func (r *grpcResolver) watcher() {
	if r.t != nil {
		panic("time already runned")
	}
	if r.freq == 0 {
		r.freq = defaultRefreshInterval
	}
	r.t = time.NewTicker(r.freq)
	for {
		select {
		case <-r.t.C:
		case <-r.ctx.Done():
			return
		}
		if err := r.refreshConnection(); err != nil {
			log.Println(err)
		}
	}
}

func (r *grpcResolver) refreshConnection() (err error) {
	var (
		service  = r.serviceName
		balancer = r.balancer
		address  []resolver.Address
	)
	if balancer == nil {
		balancer = net_balancer.Default()
	}

	backends := balancer.Backends(service)
	for _, backend := range backends {
		address = append(address, resolver.Address{
			Addr: backend.Address(),
			Metadata: &grpcMetadata{
				serviceName:          service,
				backend:              backend,
				balancer:             balancer,
				maxRequestsByBackend: r.maxRequestsByBackend,
			},
		})
	}

	r.cc.NewAddress(address)
	return err
}

var _ resolver.Resolver = (*grpcResolver)(nil)
