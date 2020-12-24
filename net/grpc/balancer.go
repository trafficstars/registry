package grpc

import (
	"sync/atomic"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"

	netbalancer "github.com/trafficstars/registry/net/balancer"
)

// NewBalancerBuilder creates a new registry balancer builder.
func NewBalancerBuilder(name string) balancer.Builder {
	return base.NewBalancerBuilder(name, &registryPickerBuilder{}, base.Config{HealthCheck: true})
}

type registryPickerBuilder struct{}

func (*registryPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	grpclog.Infof("registryPicker: newPicker called with readySCs: %v", info.ReadySCs)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	if len(info.ReadySCs) == 1 {
		for sc := range info.ReadySCs {
			return &simplePicker{subConn: sc}
		}
	}

	picker := &registryPicker{
		subConns: map[string]balancer.SubConn{},
	}

	for sc, scInfo := range info.ReadySCs {
		switch meta := scInfo.Address.Metadata.(type) {
		case nil:
		case *grpcMetadata:
			if meta.balancer != nil && picker.balancer == nil {
				picker.balancer = meta.balancer
				picker.serviceName = meta.serviceName
				picker.servicePort = meta.servicePort
				picker.maxRequestsByBackend = meta.maxRequestsByBackend
			}
			picker.subConns[scInfo.Address.Addr] = sc
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

func (p *registryPicker) Pick(opts balancer.PickInfo) (balancer.PickResult, error) {
	if p.balancer != nil {
		if backend, err := p.balancer.Next(p.serviceName, p.maxRequestsByBackend); err == nil {
			address := backend.Address()
			if p.servicePort != "" {
				address = backend.Hostname() + ":" + p.servicePort
			}
			if conn, ok := p.subConns[address]; ok {
				backend.IncConcurrentRequest(1)
				return balancer.PickResult{
					SubConn: conn,
					Done:    func(balancer.DoneInfo) { backend.IncConcurrentRequest(-1) },
				}, nil
			}
		}
	}
	next := atomic.AddUint32(&p.next, 1) % uint32(len(p.subConnList))
	sc := p.subConnList[next]
	return balancer.PickResult{
		SubConn: sc,
		Done:    nil,
	}, nil
}

type simplePicker struct {
	subConn balancer.SubConn
}

func (p *simplePicker) Pick(opts balancer.PickInfo) (balancer.PickResult, error) {
	return balancer.PickResult{
		SubConn: p.subConn,
		Done:    nil,
	}, nil
}
