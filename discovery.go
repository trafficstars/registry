package registry

import (
	"github.com/hashicorp/consul/api"

	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
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
			Interval: options.Check.Interval,
			Timeout:  options.Check.Timeout,
			DeregisterCriticalServiceAfter: "10m",
			HTTP: options.Check.HTTP,
			TCP:  options.Check.TCP,
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

func (d *discovery) Lookup(filter *Filter) ([]Service, error) {
	var result []Service
	list, _, err := d.catalog.Services(nil)
	if err != nil {
		return nil, err
	}
	var names []string
	for name := range list {
		names = append(names, name)
		items, _, err := d.catalog.Service(name, "", nil)
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
			if srv.test(filter) {
				result = append(result, srv)
			}
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
		healthChecks, _, err := d.health.Checks(name, nil)
		if err != nil {
			return nil, err
		}
		for _, check := range healthChecks {
			if i := sort.Search(len(result), func(i int) bool { return result[i].ID >= check.ServiceID }); i < len(result) && result[i].ID == check.ServiceID {
				result[i].Status = status(check.Status)
			}
		}
	}
	return result, nil
}

func dc(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "DC=") {
			return strings.TrimPrefix(tag, "DC=")
		}
	}
	return ""
}
