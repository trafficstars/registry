package registry

import (
	"github.com/hashicorp/consul/api"

	"net/url"
	"strconv"
	"time"
)

var args []string

const REGISTRY_PREFIX = "registry"

func New(dsn string, osArgs []string) (Registry, error) {
	args = osArgs
	url, err := url.Parse(dsn)

	if err != nil {
		return nil, err
	}
	client, err := api.NewClient(&api.Config{
		Scheme:     url.Scheme,
		Address:    url.Host,
		Datacenter: url.Query().Get("dc"),
		Token:      url.Query().Get("token"),
	})
	if err != nil {
		return nil, err
	}
	registry := registry{
		client:          client,
		datacenter:      url.Query().Get("dc"),
		refreshInterval: 30 * time.Second,
		bindChan:        make(chan struct{}),
	}
	if interval := url.Query().Get("refresh_interval"); len(interval) != 0 {
		if v, err := strconv.ParseInt(interval, 10, 64); err == nil && v > 0 {
			registry.refreshInterval = time.Duration(v) * time.Second
		}
	}
	if len(dsn) != 0 {
		go registry.supervisor()
	} else {
		registry.refreshInterval = -1
	}
	return &registry, nil
}

type registry struct {
	client          *api.Client
	configs         []config
	refreshInterval time.Duration
	datacenter      string
	bindChan        chan struct{}
}

func (r *registry) KV() KV {
	return &kv{client: r.client.KV()}
}

func (r *registry) Discovery() Discovery {
	return &discovery{
		agent:      r.client.Agent(),
		health:     r.client.Health(),
		catalog:    r.client.Catalog(),
		datacenter: r.datacenter,
	}
}

func (r *registry) supervisor() {
	var (
		refresh = time.Tick(r.refreshInterval)
	)
	for {
		select {
		case <-refresh:
			r.refresh()
		case <-r.bindChan:
			r.refresh()
		}
	}
}

func (r *registry) refresh() {
	var (
		kv   = r.KV()
		keys []string
	)
	for _, config := range r.configs {
		for _, item := range config.items {
			keys = append(keys, item.key)
		}
	}
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, err := kv.Get(key); err == nil {
			values[key] = v
		}
	}
	for _, config := range r.configs {
		var updatedItemKeys []string

		config.rawConfig.Lock()
		for _, item := range config.items {
			value, ok := values[item.key]
			if !ok {
				continue
			}
			if item.equal(value) {
				continue
			}
			item.set(value)
			updatedItemKeys = append(updatedItemKeys, item.path)
		}
		config.rawConfig.Unlock()

		config.callOnUpdatedMethod(updatedItemKeys) // Call method "OnUpdate<variableName>" if exists
	}
}
