package main

import (
	"github.com/trafficstars/registry"

	"fmt"
	"os"
	"sync"
	"time"
)

type Config struct {
	sync.RWMutex
	ID          int    `default:"42"           registry:"service/id"`
	Name        string `default:"Name"         registry:"service/name"`
	Description string `flag:"desc" env:"DESC" registry:"service/description"`
}

func (c *Config) GetID() int {
	c.RLock()
	defer c.RUnlock()
	return c.ID
}

func (c *Config) GetName() string {
	c.RLock()
	defer c.RUnlock()
	return c.Name
}

func main() {

	var (
		config = Config{
			Description: "Description",
		}
		registry, err = registry.New("http://127.0.0.1:8500?dc=adv&refresh_interval=5", os.Args)
	)

	if err != nil {
		panic(err)
	}

	registry.Bind(&config)

	for {
		fmt.Println(config.GetID(), config.GetName(), config.Description)
		time.Sleep(2 * time.Second)
	}
	// create new keys in consul: registry/service/id and registry/service/name and set new values
}
