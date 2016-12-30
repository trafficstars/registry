package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/trafficstars/registry"
)

const (
	defaultDockerHost       = "unix:///var/run/docker.sock"
	defaultDockerApiVersion = "1.24"
	defaultRegistryDSN      = "http://127.0.0.1:8500?dc=dc1&refresh_interval=5"
)

var hostname, _ = os.Hostname()

func Run() {
	client, err := client.NewClient(
		env("DOCKER_HOST", defaultDockerHost),
		env("DOCKER_API_VERSION", defaultDockerApiVersion),
		nil,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	registry, err := registry.New(env("REGISTRY_DSN", defaultRegistryDSN), []string{})
	if err != nil {
		log.Fatal(err)
	}
	info, err := client.Info(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	(&supervisor{
		info:      info,
		docker:    client,
		discovery: registry.Discovery(),
	}).run()
}

type Stats struct {
	CPUUsage    float64
	MemoryUsage uint64
	MemoryLimit uint64
}

type supervisor struct {
	info       types.Info
	mutex      sync.RWMutex
	docker     *client.Client
	discovery  registry.Discovery
	inProgress bool
}

func (s *supervisor) refresh() {
	if s.isInProgress() {
		return
	}
	containers, err := s.docker.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Errorf("Refresh services (container list): %v", err)
		return
	}
	s.setInProgress(true)
	for _, container := range containers {
		log.Debugf("Refresh container: %s", container.ID[:12])
		if err := s.serviceRegister(container.ID); err != nil {
			log.Errorf("Register service [%s]: %v", container.ID[:12], err)
		}
	}
	s.setInProgress(false)
}

func (s *supervisor) containerStats(containerID string) (Stats, error) {
	var (
		stats         types.Stats
		response, err = s.docker.ContainerStats(context.Background(), containerID, false)
	)
	if err != nil {
		return Stats{}, err
	}
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return Stats{}, err
	}
	var (
		cpuUsage    float64
		cpuDelta    = float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta = float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	)
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuUsage = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return Stats{
		CPUUsage:    cpuUsage,
		MemoryUsage: stats.MemoryStats.Usage,
		MemoryLimit: stats.MemoryStats.Limit,
	}, nil
}

func (s *supervisor) serviceRegister(containerID string) error {
	container, err := s.docker.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return err
	}

	if container.State.Status != "running" {
		log.Debugf("Container [%s] is not running", containerID[:12])
		return nil
	}

	var (
		name    string
		tags    []string
		address string
		hostIP  = os.Getenv("HOST_IP")
	)

	for _, env := range container.Config.Env {
		switch {
		case strings.HasPrefix(env, "SERVICE_NAME="):
			name = strings.TrimPrefix(env, "SERVICE_NAME=")
		case strings.HasPrefix(env, "TAG_"), strings.HasPrefix(env, "SERVICE_"):
			tags = append(tags, env)
		}
	}

	if len(name) == 0 {
		log.Debugf("Container [%s] is not the service", containerID[:12])
		return nil
	}

	stats, err := s.containerStats(containerID)
	if err != nil {
		return err
	}

	for _, mapping := range container.NetworkSettings.Ports {
		if len(mapping) != 0 {
			host := hostIP
			if len(mapping[0].HostIP) != 0 && mapping[0].HostIP != "0.0.0.0" {
				host = mapping[0].HostIP
			}
			address = net.JoinHostPort(host, mapping[0].HostPort)
			break
		}
	}

	tags = append(tags,
		fmt.Sprintf("HOST_IP=%s", hostIP),
		fmt.Sprintf("CPU_USAGE=%f", stats.CPUUsage),
		fmt.Sprintf("MEMORY_USAGE=%f", (float64(stats.MemoryUsage)/float64(stats.MemoryLimit))*100),
		fmt.Sprintf("MEMORY_LIMIT=%d", stats.MemoryLimit),
		fmt.Sprintf("MEMORY_TOTAL=%d", s.info.MemTotal),
		fmt.Sprintf("PORT_MAP=%s", toJson(container.NetworkSettings.Ports)),
		fmt.Sprintf("REGISTRY=%s", hostname),
	)

	return s.discovery.Register(registry.ServiceOptions{
		ID:      container.ID,
		Name:    name,
		Address: address,
		Tags:    tags,
		Check:   checkOptions(address, container.Config.Env),
	})
}

func (s *supervisor) api() {
	http.HandleFunc("/api/v1/check", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(rw)
		encoder.SetIndent("", "    ")
		if info, err := s.docker.Info(context.Background()); err == nil {
			encoder.Encode(map[string]interface{}{
				"ID":                info.ID,
				"Gorutines":         runtime.NumGoroutine(),
				"Containers":        info.Containers,
				"ContainersRunning": info.ContainersRunning,
				"ContainersPaused":  info.ContainersPaused,
				"ContainersStopped": info.ContainersStopped,
				"Images":            info.Images,
				"SystemTime":        info.SystemTime,
				"KernelVersion":     info.KernelVersion,
				"OperatingSystem":   info.OperatingSystem,
				"NCPU":              info.NCPU,
				"MemTotal":          info.MemTotal,
			})
			return
		}
		encoder.Encode(map[string]interface{}{
			"Gorutines": runtime.NumGoroutine(),
		})
	})
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func (s *supervisor) setInProgress(status bool) {
	s.mutex.Lock()
	s.inProgress = status
	s.mutex.Unlock()
}

func (s *supervisor) isInProgress() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.inProgress
}

func (s *supervisor) run() {
	var (
		tick           = time.Tick(10 * time.Second)
		events, errors = s.docker.Events(context.Background(), types.EventsOptions{})
	)
	log.Info("Run supervisor")
	s.refresh()
	go s.api()
	for {
		select {
		case event := <-events:
			switch event.Action {
			case "start", "unpause":
				log.Debugf("Register new container: %s", event.Actor.ID[:12])
				go func() {
					if err := s.serviceRegister(event.Actor.ID); err != nil {
						log.Errorf("Register service [%s]: %v", event.Actor.ID[:12], err)
					}
				}()
			case "die", "kill", "stop", "pause", "oom":
				log.Debugf("Deregister service [%s]: %s (%v)", event.Action, event.Actor.ID[:12], event.Actor.Attributes)
				if err := s.discovery.Deregister(event.Actor.ID); err != nil {
					log.Errorf("Deregister service [%s]: %v", event.Action, err)
				}
			}
		case error := <-errors:
			log.Errorf("Event: %v", error)
		case <-tick:
			go s.refresh()
		}
	}
}

func checkOptions(address string, env []string) registry.CheckOptions {
	options := registry.CheckOptions{
		Interval: "5s",
		Timeout:  "2s",
	}
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "CHECK_INTERVAL="):
			options.Interval = strings.TrimPrefix(e, "CHECK_INTERVAL=")
		case strings.HasPrefix(e, "CHECK_TIMEOUT="):
			options.Timeout = strings.TrimPrefix(e, "CHECK_TIMEOUT=")
		case strings.HasPrefix(e, "CHECK_HTTP="):
			options.HTTP = strings.Replace(strings.TrimPrefix(e, "CHECK_HTTP="), "{{address}}", address, 1)
		case strings.HasPrefix(e, "CHECK_TCP="):
			options.TCP = strings.Replace(strings.TrimPrefix(e, "CHECK_TCP="), "{{address}}", address, 1)
		}
	}
	return options
}

func env(key, defaultValue string) string {
	if value := os.Getenv(key); len(value) != 0 {
		return value
	}
	return defaultValue
}

func toJson(v interface{}) string {
	if json, err := json.Marshal(v); err == nil {
		return string(json)
	}
	return ""
}
