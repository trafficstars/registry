package balancer

import (
	"net"
)

func listOfLocalAddresses() (address []string, err error) {
	var ifaces []net.Interface
	if ifaces, err = net.Interfaces(); err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // Skip all down interfaces
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPAddr:
				ip = v.IP
			case *net.IPNet:
				ip = v.IP
			}
			if ip != nil {
				address = append(address, ip.String())
			}
		}
	}

	return
}
