package grpc

import (
	"google.golang.org/grpc/resolver"
)

// ipResolver watches for the name resolution update for an IP address.
type ipResolver struct {
	cc resolver.ClientConn
	ip []resolver.Address
}

// ResolveNow resend the address it stores, no resolution is needed.
func (i *ipResolver) ResolveNow(opt resolver.ResolveNowOption) {
	i.cc.NewAddress(i.ip)
}

// Close closes the ipResolver.
func (i *ipResolver) Close() {}

var _ resolver.Resolver = (*ipResolver)(nil)
