package registry

import (
	"github.com/hashicorp/consul/api"

	"net"
	"net/url"
	"strconv"
	"strings"
)

type discovery struct {
	agent      *api.Agent
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

func (d *discovery) Lookup(filter *Filter) ([]Service, error) {
	statuses := make(map[string]int8)
	if checks, err := d.agent.Checks(); err == nil {
		for _, check := range checks {
			status := SERVICE_STATUS_UNDEFINED
			switch check.Status {
			case "passing":
				status = SERVICE_STATUS_PASSING
			case "warning":
				status = SERVICE_STATUS_WARNING
			case "critical":
				status = SERVICE_STATUS_CRITICAL
			}
			statuses[check.ServiceID] = status
		}
	}
	var (
		result        = make([]Service, 0, len(statuses))
		services, err = d.agent.Services()
	)
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		if service.Service == "consul" {
			continue
		}
		status := SERVICE_STATUS_UNDEFINED
		if s, ok := statuses[service.ID]; ok {
			status = s
		}
		var (
			srv = Service{
				ID:         service.ID,
				Name:       service.Service,
				Datacenter: dc(service.Tags),
				Address:    service.Address,
				Port:       service.Port,
				Tags:       service.Tags,
				Status:     status,
			}
		)
		if srv.test(filter) {
			result = append(result, srv)
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
