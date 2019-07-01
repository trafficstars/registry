package http

import "testing"

// goos: darwin
// goarch: amd64
// pkg: gotask.ws/scm/ts/rotator/vendor/github.com/trafficstars/registry/net/http
// Benchmark_NextBackend/nextBackend-4         	 3000000	       539 ns/op	       0 B/op	       0 allocs/op
// Benchmark_NextBackend/nextWeightBackend-4   	 2000000	       655 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	gotask.ws/scm/ts/rotator/vendor/github.com/trafficstars/registry/net/http	5.124s
func Benchmark_NextBackend(b *testing.B) {
	ups := upstream{
		backends: backends{
			&backend{address: "test1"},
			&backend{address: "test2"},
			&backend{address: "test3"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("nextBackend", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = ups.nextBackend(0)
			}
		})
	})

	b.Run("nextWeightBackend", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = ups.nextWeightBackend(0)
			}
		})
	})
}
