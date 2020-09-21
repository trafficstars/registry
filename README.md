# registry lookup service

[![Build Status](https://travis-ci.org/trafficstars/registry.svg?branch=master)](https://travis-ci.org/trafficstars/registry)
[![Go Report Card](https://goreportcard.com/badge/github.com/trafficstars/registry)](https://goreportcard.com/report/github.com/trafficstars/registry)
[![GoDoc](https://godoc.org/github.com/trafficstars/registry?status.svg)](https://godoc.org/github.com/trafficstars/registry)
[![Coverage Status](https://coveralls.io/repos/github/trafficstars/registry/badge.svg)](https://coveralls.io/github/trafficstars/registry)

Library provides the interface and service discovery based on abstract discovery interface.

Supports:

* [X] Consul
* [ ] Zookeeper
* [ ] etcd

## GRPC configuration

```go
import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"

	"github.com/trafficstars/registry"
	registry_balancer "github.com/trafficstars/registry/net/balancer"
	grpc_transport "github.com/trafficstars/registry/net/grpc"
)

func main() {
	// Init registry & service descovery
	registry, err := registry.New(registryDSN, registryArgs)
	if err != nil {
		log.Fatal(err)
	}

	// Init global network load-balancer
	registry_balancer.Init(registry_balancer.RoundRobinStrategy, myRegistry.Discovery())

	// Register balancer and connection resolver
	balancer.Register(grpc_transport.NewBalancerBuilder("registry"))
	resolver.Register(grpc_transport.NewResolveBuilder("registry", myRegistry.Discovery()))
	resolver.SetDefaultScheme("registry")
}
```
