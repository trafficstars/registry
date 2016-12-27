package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/trafficstars/registry"
	transport "github.com/trafficstars/registry/net/http"
)

type service struct {
	name string
}

func (s *service) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("srv", s.name)
	fmt.Fprintf(rw, "I'm alive")
}

func main() {
	r, err := registry.New("http://127.0.0.1:8500?dc=adv&refresh_interval=5", os.Args)
	if err != nil {
		panic(err)
	}
	discovery := r.Discovery()

	go func() {
		discovery.Register(registry.ServiceOptions{
			ID:      "serviceID",
			Name:    "example-service",
			Address: "127.0.0.1:8888",
			Tags:    []string{"A", "B", "C", "SERVICE_WEIGHT=3"},
			Check: registry.CheckOptions{
				Interval: "5s",
				Timeout:  "2s",
				HTTP:     "http://127.0.0.1:8888/api/v1/check",
			},
		})
		http.ListenAndServe(":8888", &service{"srv1"})
	}()

	go func() {
		discovery.Register(registry.ServiceOptions{
			ID:      "serviceID2",
			Name:    "example-service",
			Address: "127.0.0.1:8889",
			Tags:    []string{"A", "B", "C", "SERVICE_WEIGHT=10"},
			Check: registry.CheckOptions{
				Interval: "5s",
				Timeout:  "2s",
				HTTP:     "http://127.0.0.1:8889/api/v1/check",
			},
		})
		http.ListenAndServe(":8889", &service{"srv2"})
	}()

	go func() {
		discovery.Register(registry.ServiceOptions{
			ID:      "serviceID3",
			Name:    "example-service",
			Address: "127.0.0.1:8899",
			Tags:    []string{"A", "B", "C", "SERVICE_WEIGHT=7"},
			Check: registry.CheckOptions{
				Interval: "5s",
				Timeout:  "2s",
				HTTP:     "http://127.0.0.1:8899/api/v1/check",
			},
		})
		http.ListenAndServe(":8899", &service{"srv3"})
	}()

	transport.Init(discovery)

	client := http.Client{
		Transport: &transport.Transport{
			Transport: http.Transport{
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     10 * time.Second,
				// etc ...
			},
		},
	}

	tick := time.Tick(time.Second)
	for {
		<-tick
		for i := 0; i < 2; i++ {
			response, err := client.Get("http://example-service/api/v1/check")
			if err != nil {
				fmt.Println(err)
				continue
			}
			body, _ := ioutil.ReadAll(response.Body)
			fmt.Println(response.Header.Get("srv"), string(body))
		}
		fmt.Println("================")
	}
}
