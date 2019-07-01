package http

import "sync/atomic"

type upstream struct {
	// Current backend index
	index uint32

	// Balancing parameters
	currentWeight int32
	maxWeight     int32

	// priority backend
	priorityBackend *backend

	// List of the upstream backends
	backends backends

	// greatest common divisor
	gcd int32
}

func (ups *upstream) nextBackend(maxRequestsByBackend int) (back *backend) {
	// First send requests to the priority backend (generally this is the local service)
	if ups.priorityBackend != nil {
		if maxRequestsByBackend <= 0 || maxRequestsByBackend > ups.priorityBackend.concurrentRequestCount() {
			return ups.priorityBackend
		}
	}

	backends := ups.backends
	backendCount := uint32(len(backends))

	for i := uint32(0); i < backendCount; i++ {
		index := atomic.AddUint32(&ups.index, 1)
		back = backends[index%backendCount]

		if maxRequestsByBackend <= 0 || maxRequestsByBackend > back.concurrentRequestCount() {
			return back
		}
	}
	return nil
}

func (ups *upstream) nextWeightBackend(maxRequestsByBackend int) *backend {
	// First send requests to the priority backend (generally this is the local service)
	if ups.priorityBackend != nil {
		if maxRequestsByBackend <= 0 || maxRequestsByBackend > ups.priorityBackend.concurrentRequestCount() {
			return ups.priorityBackend
		}
	}

	backends := ups.backends
	backendCount := uint32(len(backends))
	if backendCount < 1 {
		return nil
	}

	for i := uint32(0); i < backendCount; i++ {
		var (
			currentWeight int32
			index         = atomic.AddUint32(&ups.index, 1) % backendCount
			backend       = backends[index]
		)

		if maxRequestsByBackend > backend.concurrentRequestCount() {
			continue
		}

		if index == 0 {
			currentWeight = atomic.AddInt32(&ups.currentWeight, -ups.gcd)
			if currentWeight <= 0 {
				atomic.StoreInt32(&ups.currentWeight, ups.maxWeight)
				if ups.maxWeight == 0 {
					return backend
				}
				currentWeight = ups.maxWeight
			}
		} else {
			currentWeight = atomic.LoadInt32(&ups.currentWeight)
		}

		if int32(backend.weight) >= currentWeight {
			if backend.doSkip() {
				continue
			}
			return backend
		}
	}
	return nil
}
