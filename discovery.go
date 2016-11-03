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
		v, err := strconv.ParseInt(p, 10, 16)
		if err != nil {
			return err
		}
		host = h
		port = int(v)
	}
	return d.agent.ServiceRegister(&api.AgentServiceRegistration{
		ID:                options.ID + "::" + options.Name + "::" + d.datacenter,
		Name:              options.Name,
		Address:           host,
		Port:              port,
		Tags:              options.Tags,
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
			id, datacenter = splitID(service.ID)
			srv            = Service{
				ID:         id,
				Name:       service.Service,
				Datacenter: datacenter,
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

func splitID(id string) (string, string) {
	if parts := strings.Split(id, "::"); len(parts) == 3 {
		return parts[0], parts[2]
	}
	return id, ""
}
