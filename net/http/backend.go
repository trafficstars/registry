package http

import "sync/atomic"

type backend struct {
	skipCounter    int32
	requestCounter int32
	weight         int32
	hostaddress    string
	address        string
}

type backends []*backend

func (b *backend) skip() {
	atomic.StoreInt32(&b.skipCounter, 7)
}

func (b *backend) doSkip() bool {
	return atomic.AddInt32(&b.skipCounter, -1) >= 0
}

func (b *backend) requestCount() int {
	return (int)(atomic.LoadInt32(&b.requestCounter))
}

func (b *backend) incRequest(v int32) int32 {
	return atomic.AddInt32(&b.requestCounter, v)
}

func (b backends) maxWeight() int32 {
	maxWeight := int32(-1)
	for _, backend := range b {
		if backend.weight > maxWeight {
			maxWeight = backend.weight
		}
	}
	return maxWeight
}

func (b backends) gcd() int32 {
	divisor := int32(-1)
	for _, backend := range b {
		if divisor == -1 {
			divisor = backend.weight
		} else {
			divisor = gcd(divisor, backend.weight)
		}
	}
	return divisor
}

func gcd(a, b int32) int32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
