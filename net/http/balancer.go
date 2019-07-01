package http

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/trafficstars/registry"
)

// BalancingStrategy type
type BalancingStrategy int

// Balancyng strategy variants
const (
	RoundRobinStrategy BalancingStrategy = iota
	WeightStrategy
)

var _balancer balancer

// Init balancer based on descovery
func Init(strategy BalancingStrategy, discovery registry.Discovery, localAddrs ...string) {
	if len(localAddrs) == 0 || localAddrs[0] == "" {
		localAddrs, _ = listOfLocalAddresses()

		// all IPv4 addresses on the local machine
		const allAddress = "0.0.0.0"
		hasAllAddress := false
		for _, addr := range localAddrs {
			if addr == allAddress {
				hasAllAddress = true
				break
			}
		}
		if !hasAllAddress {
			localAddrs = append(localAddrs, allAddress)
		}
	}

	_balancer = balancer{
		strategy:   strategy,
		discovery:  discovery,
		localAddrs: localAddrs,
	}

	upstreams := make(map[string]*upstream)
	atomic.StorePointer(&_balancer.upstreams, unsafe.Pointer(&upstreams))

	_balancer.lookup()
	go _balancer.supervisor()
}

type balancer struct {
	// Strategy of address balancing
	strategy BalancingStrategy

	// upstreams map[string]*upstream
	upstreams unsafe.Pointer
	discovery registry.Discovery

	localAddrs []string
}

func (b *balancer) lookup() error {
	var (
		backendServices = map[string]backends{}
		services, err   = b.discovery.Lookup(nil)
	)
	if err != nil {
		return err
	}

	// Group backends by services
	for _, service := range services {
		if service.Status == registry.SERVICE_STATUS_PASSING {
			backendServices[service.Name] = append(backendServices[service.Name], &backend{
				weight:      int32(serverWeight(&service)),
				hostaddress: service.Address,
				address:     net.JoinHostPort(service.Address, strconv.Itoa(service.Port)),
			})
		}
	}

	upstreams := map[string]*upstream{}

	for key, backends := range backendServices {
		var priorityBackend *backend

	loop:
		for _, bk := range backends {
			for _, addr := range b.localAddrs {
				if addr == bk.hostaddress {
					priorityBackend = bk
					break loop
				}
			}
		}

		upstreams[key] = &upstream{
			priorityBackend: priorityBackend,
			backends:        backends,
			gcd:             backends.gcd(),
			maxWeight:       backends.maxWeight(),
		}
	}

	atomic.StorePointer(&b.upstreams, unsafe.Pointer(&upstreams))

	return nil
}

func (b *balancer) supervisor() {
	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-tick:
			b.lookup()
		}
	}
}

func (b *balancer) countOfBackends(service string) int {
	upstreams := *(*map[string]*upstream)(atomic.LoadPointer(&b.upstreams))
	if upstream, found := upstreams[service]; found {
		return len(upstream.backends)
	}
	return 0
}

func (b *balancer) nextRoundRobin(service string, maxRequestsByBackend int) (*backend, error) {
	upstream := b.getUpstreamByServiceName(service)
	if upstream != nil {
		return upstream.nextBackend(maxRequestsByBackend), nil
	}
	return nil, fmt.Errorf("Service '%s' not found", service)
}

func (b *balancer) nextWeight(service string, maxRequestsByBackend int) (*backend, error) {
	upstream := b.getUpstreamByServiceName(service)
	if upstream == nil {
		return nil, fmt.Errorf("Service '%s' not found", service)
	}

	if backend := upstream.nextWeightBackend(maxRequestsByBackend); backend != nil {
		return backend, nil
	}

	return nil, fmt.Errorf("Service backend of '%s' not found", service)
}

func (b *balancer) next(service string, maxRequestsByBackend int) (*backend, error) {
	if b.strategy == WeightStrategy {
		return b.nextWeight(service, maxRequestsByBackend)
	}
	return b.nextRoundRobin(service, maxRequestsByBackend)
}

func (b *balancer) getUpstreamByServiceName(service string) *upstream {
	upstreams := *(*map[string]*upstream)(atomic.LoadPointer(&b.upstreams))
	ups, _ := upstreams[service]
	return ups
}

func serverWeight(s *registry.Service) int {
	weight := 1
	for _, tag := range s.Tags {
		if strings.HasPrefix(tag, "SERVICE_WEIGHT=") {
			if v, _ := strconv.ParseInt(strings.TrimPrefix(tag, "SERVICE_WEIGHT="), 10, 64); v != 0 {
				weight = int(v)
			}
		}
	}
	weight *= 100
	for _, tag := range s.Tags {
		if strings.HasPrefix(tag, "CPU_USAGE=") {
			if v, _ := strconv.ParseFloat(strings.TrimPrefix(tag, "CPU_USAGE="), 64); v != 0 {
				if usage := int(math.Ceil(v / 4.0)); usage != 0 {
					weight = weight / int(math.Ceil(v/4.0))
				}
			}
		}
	}
	return weight
}
