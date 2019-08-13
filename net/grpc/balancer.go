package grpc

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"

	netbalancer "github.com/trafficstars/registry/net/balancer"
)

// NewBalancerBuilder creates a new registry balancer builder.
func NewBalancerBuilder(name string) balancer.Builder {
	return base.NewBalancerBuilderWithConfig(name, &registryPickerBuilder{}, base.Config{HealthCheck: true})
}

type registryPickerBuilder struct{}

func (*registryPickerBuilder) Build(readySCs map[resolver.Address]balancer.SubConn) balancer.Picker {
	grpclog.Infof("registryPicker: newPicker called with readySCs: %v", readySCs)
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	if len(readySCs) == 1 {
		for _, sc := range readySCs {
			return &simplePicker{subConn: sc}
		}
	}

	picker := &registryPicker{
		subConns: map[string]balancer.SubConn{},
	}

	for addr, sc := range readySCs {
		switch meta := addr.Metadata.(type) {
		case nil:
		case *grpcMetadata:
			if meta.balancer != nil && picker.balancer == nil {
				picker.balancer = meta.balancer
				picker.serviceName = meta.serviceName
				picker.servicePort = meta.servicePort
				picker.maxRequestsByBackend = meta.maxRequestsByBackend
			}
			picker.subConns[addr.Addr] = sc
		}
		picker.subConnList = append(picker.subConnList, sc)
	}

	return picker
}

type registryPicker struct {
	next                 uint32
	serviceName          string
	servicePort          string
	balancer             netbalancer.Balancer
	subConns             map[string]balancer.SubConn
	subConnList          []balancer.SubConn
	maxRequestsByBackend int
}

func (p *registryPicker) Pick(ctx context.Context, opts balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	if p.balancer != nil {
		if backend, err := p.balancer.Next(p.serviceName, p.maxRequestsByBackend); err == nil {
			address := backend.Address()
			if p.servicePort != "" {
				address = backend.Hostname() + ":" + p.servicePort
			}
			if conn, ok := p.subConns[address]; ok {
				backend.IncConcurrentRequest(1)
				return conn, func(balancer.DoneInfo) { backend.IncConcurrentRequest(-1) }, nil
			}
		}
	}
	next := atomic.AddUint32(&p.next, 1) % uint32(len(p.subConnList))
	sc := p.subConnList[next]
	return sc, nil, nil
}

type simplePicker struct {
	subConn balancer.SubConn
}

func (p *simplePicker) Pick(ctx context.Context, opts balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	return p.subConn, nil, nil
}
