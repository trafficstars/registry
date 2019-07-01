package http

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

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
		upstreams:  make(map[string]*upstream),
		discovery:  discovery,
		localAddrs: localAddrs,
	}
	_balancer.lookup()
	go _balancer.supervisor()
}

type balancer struct {
	mutex sync.RWMutex

	// Strategy of address balancing
	strategy BalancingStrategy

	upstreams map[string]*upstream
	backends  map[string]backends
	discovery registry.Discovery

	localAddrs []string
}

func (b *balancer) lookup() error {
	var (
		backendServices = make(map[string]backends, len(b.backends))
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

	b.mutex.Lock()
	defer b.mutex.Unlock()

	for key := range b.upstreams {
		if _, found := backendServices[key]; !found {
			delete(b.upstreams, key)
		}
	}

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

		b.upstreams[key] = &upstream{
			priorityBackend: priorityBackend,
			backends:        backends,
			gcd:             backends.gcd(),
			maxWeight:       backends.maxWeight(),
		}
	}
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
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if upstream, found := b.upstreams[service]; found {
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
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	ups, _ := b.upstreams[service]
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
