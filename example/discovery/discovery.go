package main

import (
	"github.com/trafficstars/registry"

	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	r, err := registry.New("http://127.0.0.1:8500?dc=dc1&refresh_interval=5", os.Args)
	if err != nil {
		panic(err)
	}

	discovery := r.Discovery()
	discovery.Register(registry.ServiceOptions{
		ID:      fmt.Sprint(time.Now().Unix()),
		Name:    "example_service",
		Address: "127.0.0.1:8888",
		Tags:    []string{"A", "B", "C"},
		Check: registry.CheckOptions{
			Interval: "5s",
			Timeout:  "2s",
			HTTP:     "http://127.0.0.1:8888/check",
		},
	})

	go func() {
		tick := time.Tick(2 * time.Second)
		for {
			<-tick
			if services, err := discovery.Lookup(&registry.Filter{Service: "example_service", Status: registry.SERVICE_STATUS_PASSING}); err == nil {
				for _, service := range services {
					fmt.Printf("%#v %d\n", service, registry.SERVICE_STATUS_PASSING)
				}
			}
		}
	}()

	http.HandleFunc("/check", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "I'm alive")
	})
	http.ListenAndServe(":8888", nil)
}
