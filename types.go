package registry

import (
	"sync"
)

const (
	SERVICE_STATUS_UNDEFINED int8 = iota
	SERVICE_STATUS_PASSING
	SERVICE_STATUS_WARNING
	SERVICE_STATUS_CRITICAL
)

type Registry interface {
	KV() KV
	Bind(i sync.Locker) error
	Discovery() Discovery
}

type KV interface {
	Get(string) (string, error)
	Set(key, value string) error
	List(prefix string) (map[string]string, error)
	Delete(string) error
}

type Discovery interface {
	Lookup(*Filter) ([]Service, error)
	Register(ServiceOptions) error
	Deregister(string) error
}

type Service struct {
	ID         string
	Name       string
	Datacenter string
	Address    string
	Port       int
	Tags       []string
	Status     int8
}

func (s *Service) test(filter *Filter) bool {
	if filter == nil {
		return true
	}
	if len(filter.Datacenter) != 0 && filter.Datacenter != s.Datacenter {
		return false
	}
	if len(filter.Service) != 0 && filter.Service != s.Name {
		return false
	}
	if len(filter.Tags) != 0 {
		for _, ft := range filter.Tags {
			for _, st := range s.Tags {
				if ft == st {
					return true
				}
			}
		}
		return false
	}
	return true
}

type Filter struct {
	Tags       []string
	Service    string
	Datacenter string
}

type ServiceOptions struct {
	ID      string
	Name    string
	Address string
	Tags    []string
	Check   CheckOptions
}
type CheckOptions struct {
	Interval string
	Timeout  string
	HTTP     string
	TCP      string
}
