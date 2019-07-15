package registry

import (	
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
)

type discovery struct {
	agent      *api.Agent
	health     *api.Health
	catalog    *api.Catalog
	datacenter string
}

func (d *discovery) Register(options ServiceOptions) error {
	var (
		host = options.Address
		port int
	)
	if strings.HasPrefix(host, "http") {
		url, err := url.Parse(host)
		if err != nil {
			return err
		}
		host = url.Host
	}
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return err
		}
		v, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return err
		}
		host = h
		port = int(v)
	}
	return d.agent.ServiceRegister(&api.AgentServiceRegistration{
		ID:                options.ID,
		Name:              options.Name,
		Address:           host,
		Port:              port,
		Tags:              append(options.Tags, "DC="+d.datacenter),
		EnableTagOverride: true,
		Check: &api.AgentServiceCheck{
			Interval:                       options.Check.Interval,
			Timeout:                        options.Check.Timeout,
			DeregisterCriticalServiceAfter: "10m",
			HTTP:                           options.Check.HTTP,
			TCP:                            options.Check.TCP,
		},
	})
}

func (d *discovery) Deregister(ident string) error {
	return d.agent.ServiceDeregister(ident)
}

type sortServiceByID []Service

func (a sortServiceByID) Len() int           { return len(a) }
func (a sortServiceByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortServiceByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// Lookup services by filter
//
// If needed to lookup all services around all DCs,
// set DC filter to "all". It takes all DCs from discovery
// and polls services every of them
func (d *discovery) Lookup(filter *Filter) ([]Service, error) {
	if filter == nil {
		filter = &Filter{}
	}

	if filter.Datacenter != "all" {
		return d.lookup(filter)
	}

	dcl, err := d.catalog.Datacenters()
	if err != nil {
		return nil, fmt.Errorf("datacenters list: %s", err)
	}

	var services []Service
	for _, dc := range dcl {
		filter.Datacenter = dc
		s, err := d.lookup(filter)
		if err != nil {
			return nil, fmt.Errorf("datacenter %s lookup: %s", dc, err)
		}
		services = append(services, s...)
	}

	return services, nil
}

func (d *discovery) lookup(filter *Filter) ([]Service, error) {
	var (
		result []Service
		q      = &api.QueryOptions{Datacenter: filter.Datacenter}
	)
	list, _, err := d.catalog.Services(q)
	if err != nil {
		return nil, err
	}
	var names []string
	for name := range list {
		names = append(names, name)
		items, _, err := d.catalog.Service(name, "", q)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			var (
				srv = Service{
					ID:         item.ServiceID,
					Name:       item.ServiceName,
					Datacenter: dc(item.ServiceTags),
					Address:    item.ServiceAddress,
					Port:       item.ServicePort,
					Tags:       item.ServiceTags,
					Status:     SERVICE_STATUS_UNDEFINED,
				}
			)
			result = append(result, srv)
		}
	}
	sort.Sort(sortServiceByID(result))
	status := func(status string) int8 {
		switch status {
		case "passing":
			return SERVICE_STATUS_PASSING
		case "warning":
			return SERVICE_STATUS_WARNING
		case "critical":
			return SERVICE_STATUS_CRITICAL
		}
		return SERVICE_STATUS_UNDEFINED
	}
	for _, name := range names {
		healthChecks, _, err := d.health.Checks(name, q)
		if err != nil {
			return nil, err
		}
		for _, check := range healthChecks {
			if i := sort.Search(len(result), func(i int) bool { return result[i].ID >= check.ServiceID }); i < len(result) && result[i].ID == check.ServiceID {
				result[i].Status = status(check.Status)
			}
		}
	}
	services := make([]Service, 0, len(result))
	for _, srv := range result {
		if srv.test(filter) {
			services = append(services, srv)
		}
	}
	return services, nil
}

func dc(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "DC=") {
			return strings.TrimPrefix(tag, "DC=")
		}
	}
	return ""
}
