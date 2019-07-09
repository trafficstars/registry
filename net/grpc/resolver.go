package grpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc/resolver"

	"github.com/trafficstars/registry"
	"github.com/trafficstars/registry/net/balancer"
)

// BuilderOption type
type BuilderOption func(b *builder)

// Balancer reolver option
func Balancer(balancer balancer.Balancer) BuilderOption {
	return func(b *builder) {
		b.balancer = balancer
	}
}

// RefreshInterval option
func RefreshInterval(freq time.Duration) BuilderOption {
	return func(b *builder) {
		b.freq = freq
	}
}

// NewResolveBuilder of the regestry services
func NewResolveBuilder(name string, discovery registry.Discovery, opts ...BuilderOption) resolver.Builder {
	b := &builder{name: name, discovery: discovery}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

type builder struct {
	name      string
	freq      time.Duration
	discovery registry.Discovery
	balancer  balancer.Balancer
}

// Build creates a new resolver for the given target.
//
// gRPC dial calls Build synchronously, and fails if the returned error is not nil.
func (b *builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	host, port, err := parseTarget(target.Endpoint, defaultPort)
	if err != nil {
		return nil, err
	}

	// If IP address then use simple resolver
	if net.ParseIP(host) != nil {
		host, _ = formatIP(host)
		addr := []resolver.Address{{Addr: host + ":" + port}}
		i := &ipResolver{cc: cc, ip: addr}
		cc.NewAddress(addr)
		return i, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	resolv := &grpcResolver{
		serviceName: host,
		defaultPort: port,
		balancer:    b.balancer,
		freq:        b.freq,
		ctx:         ctx,
		cancel:      cancel,
		cc:          cc,
	}
	resolv.refreshConnection()
	go resolv.watcher()
	return resolv, nil
}

// Scheme returns the scheme supported by this resolver.
// Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
func (b *builder) Scheme() string {
	return b.name
}
