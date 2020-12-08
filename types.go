package registry

import "sync"

const (
	SERVICE_STATUS_UNDEFINED int8 = iota + 1
	SERVICE_STATUS_PASSING
	SERVICE_STATUS_WARNING
	SERVICE_STATUS_CRITICAL
)

// Registry functionality definition
type Registry interface {
	KV() KV
	Bind(i sync.Locker) error
	Discovery() Discovery
	Refresh()
}

// KV is key value storage functionality definition
type KV interface {
	Get(string) (string, error)
	Set(key, value string) error
	List(prefix string) (map[string]string, error)
	Delete(string) error
}

// Descovery service functionality definition
type Discovery interface {
	Lookup(*Filter) ([]Service, error)
	Register(ServiceOptions) error
	Deregister(string) error
}

// Service config definition
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
	if len(filter.ID) != 0 && filter.ID != s.ID {
		return false
	}
	if filter.Status != 0 && filter.Status != s.Status {
		return false
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

// Filter search descovery definition
type Filter struct {
	ID         string
	Status     int8
	Tags       []string
	Service    string
	Datacenter string
}

// ServiceOptions defines proxy sevice object
type ServiceOptions struct {
	ID      string
	Name    string
	Address string
	Tags    []string
	Check   CheckOptions
}

// CheckOptions defines sevice healthcheck
type CheckOptions struct {
	Interval        string
	Timeout         string
	HTTP            string
	TCP             string
	TTL             string
	DeregisterAfter string
}
