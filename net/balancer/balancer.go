package balancer

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

// Balancer implements functionality of the dynamic balancing of the backends
type Balancer interface {
	// Run balancer autolookup
	Run() error

	// CountOfBackends returns count of backends for specific service
	CountOfBackends(service string) int

	// Next returns new backend according to the strategy
	Next(service string, maxRequestsByBackend int) (*Backend, error)

	// Backends returns list of backends of the paticular service
	Backends(service string) []*Backend

	// Refresh current balancer state
	Refresh() error

	// Close current balancer
	Close() error
}

type balancer struct {
	// Strategy of address balancing
	strategy BalancingStrategy

	// upstreams map[string]*upstream
	upstreams unsafe.Pointer
	discovery registry.Discovery

	localAddrs []string
	quit       chan bool
}

// New returns new balancer interface
func New(strategy BalancingStrategy, discovery registry.Discovery, localAddrs ...string) (_ Balancer, err error) {
	if len(localAddrs) == 0 || localAddrs[0] == "" {
		if localAddrs, err = listOfLocalAddresses(); err != nil {
			return nil, err
		}
	}

	blnc := &balancer{
		strategy:   strategy,
		discovery:  discovery,
		localAddrs: localAddrs,
	}

	upstreams := make(map[string]*upstream)
	atomic.StorePointer(&blnc.upstreams, unsafe.Pointer(&upstreams))

	return blnc, nil
}

// Run balancer autolookup
func (b *balancer) Run() error {
	if err := b.lookup(); err != nil {
		return err
	}
	go b.supervisor()
	return nil
}

// CountOfBackends returns count of backends for specific service
func (b *balancer) CountOfBackends(service string) (count int) {
	upstream := b.getUpstreamByServiceName(service)
	if upstream != nil {
		count = len(upstream.backends)
		if upstream.priorityBackend != nil {
			count++
		}
	}
	return count
}

// Backends returns list of backends of the paticular service
func (b *balancer) Backends(service string) (arr []*Backend) {
	upstream := b.getUpstreamByServiceName(service)
	if upstream == nil {
		return nil
	}
	arr = ([]*Backend)(upstream.backends)
	if upstream.priorityBackend != nil {
		arr = append(arr, upstream.priorityBackend)
	}
	return arr
}

// Next returns new backend according to the strategy
func (b *balancer) Next(service string, maxRequestsByBackend int) (*Backend, error) {
	if b.strategy == WeightStrategy {
		return b.nextWeight(service, maxRequestsByBackend)
	}
	return b.nextRoundRobin(service, maxRequestsByBackend)
}

// Refresh current balancer state
func (b *balancer) Refresh() error {
	return b.lookup()
}

// Close current balancer
func (b *balancer) Close() error {
	close(b.quit)
	return nil
}

func (b *balancer) nextRoundRobin(service string, maxRequestsByBackend int) (*Backend, error) {
	upstream := b.getUpstreamByServiceName(service)
	if upstream != nil {
		return upstream.nextBackend(maxRequestsByBackend), nil
	}
	return nil, fmt.Errorf("Service '%s' not found", service)
}

func (b *balancer) nextWeight(service string, maxRequestsByBackend int) (*Backend, error) {
	upstream := b.getUpstreamByServiceName(service)
	if upstream == nil {
		return nil, fmt.Errorf("Service '%s' not found", service)
	}

	if backend := upstream.nextWeightBackend(maxRequestsByBackend); backend != nil {
		return backend, nil
	}

	return nil, fmt.Errorf("Service backend of '%s' not found", service)
}

func (b *balancer) getUpstreamByServiceName(service string) *upstream {
	upstreams := *(*map[string]*upstream)(atomic.LoadPointer(&b.upstreams))
	ups, _ := upstreams[service]
	return ups
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
		if service.Status == registry.SERVICE_STATUS_PASSING || service.Status == registry.SERVICE_STATUS_UNDEFINED {
			backendServices[service.Name] = append(backendServices[service.Name], &Backend{
				weight:      int32(serverWeight(&service)),
				hostaddress: service.Address,
				address:     net.JoinHostPort(service.Address, strconv.Itoa(service.Port)),
			})
		}
	}

	upstreams := map[string]*upstream{}

	for key, backends := range backendServices {
		var priorityBackend *Backend

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
	tick := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-tick.C:
			b.lookup()
		case <-b.quit:
			tick.Stop()
			return
		}
	}
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
