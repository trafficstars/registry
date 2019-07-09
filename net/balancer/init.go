package balancer

import (
	"github.com/trafficstars/registry"
)

var _balancer Balancer

// Init default balancer based on descovery
func Init(strategy BalancingStrategy, discovery registry.Discovery, localAddrs ...string) (err error) {
	_balancer, err = New(strategy, discovery, localAddrs...)
	if err != nil {
		return err
	}
	return _balancer.Run()
}

// Default balancer
func Default() Balancer {
	return _balancer
}

// SetDefault balancer
func SetDefault(balancer Balancer) {
	_balancer = balancer
}
