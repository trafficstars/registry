package http

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/trafficstars/registry"
)

var _balancer balancer

func Init(discovery registry.Discovery) {
	rand.Seed(time.Now().UnixNano())
	_balancer = balancer{
		upstreams: make(map[string]*upstream),
		discovery: discovery,
	}
	_balancer.lookup()
	go _balancer.supervisor()
}

type (
	backend struct {
		weight      int
		skipCounter int
		address     string
	}
	backends []*backend
	upstream struct {
		index         int
		gcd           int //greatest common divisor
		maxWeight     int
		currentWeight int
		backends      backends
	}
)

func (b *backend) skip() {
	b.skipCounter = 7
}
func (b backends) maxWeight() int {
	maxWeight := -1
	for _, backend := range b {
		if backend.weight > maxWeight {
			maxWeight = backend.weight
		}
	}
	return maxWeight
}

func (b backends) gcd() int {
	divisor := -1
	for _, backend := range b {
		if divisor == -1 {
			divisor = backend.weight
		} else {
			divisor = gcd(divisor, backend.weight)
		}
	}
	return divisor
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

type balancer struct {
	mutex     sync.RWMutex
	upstreams map[string]*upstream
	backends  map[string]backends
	discovery registry.Discovery
}

func (b *balancer) lookup() {
	var (
		backends      = make(map[string]backends, len(b.backends))
		services, err = b.discovery.Lookup(nil)
	)
	if err != nil {
		return
	}
	for _, service := range services {
		if service.Status == registry.SERVICE_STATUS_PASSING {
			backends[service.Name] = append(backends[service.Name], &backend{
				weight:  serverWeight(&service),
				address: net.JoinHostPort(service.Address, strconv.Itoa(service.Port)),
			})
		}
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for key := range b.upstreams {
		if _, found := backends[key]; !found {
			delete(b.upstreams, key)
		}
	}
	for key, backends := range backends {
		b.upstreams[key] = &upstream{
			backends:  backends,
			gcd:       backends.gcd(),
			maxWeight: backends.maxWeight(),
		}
	}
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

func (b *balancer) next(service string) (*backend, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if upstream, found := b.upstreams[service]; found {
		for {
			upstream.index = (upstream.index + 1) % len(upstream.backends)
			if upstream.index == 0 {
				upstream.currentWeight = upstream.currentWeight - upstream.gcd
				if upstream.currentWeight <= 0 {
					upstream.currentWeight = upstream.maxWeight
					if upstream.currentWeight == 0 {
						return upstream.backends[upstream.index], nil
					}
				}
			}
			if backend := upstream.backends[upstream.index]; backend.weight >= upstream.currentWeight {
				if backend.skipCounter != 0 {
					backend.skipCounter--
					continue
				}
				return backend, nil
			}
		}
	}
	return nil, fmt.Errorf("Service '%s' not found", service)
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
				weight = weight / int(math.Ceil(v/4.0))
			}
		}
	}
	return weight
}
