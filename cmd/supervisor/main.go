package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/trafficstars/registry/supervisor"

	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {
	var formatter log.Formatter = &log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05 MST",
	}
	if log.IsTerminal() {
		formatter = &log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05 MST",
		}
	}
	log.SetLevel(log.InfoLevel)
	if len(os.Getenv("DEBUG")) != 0 {
		log.SetLevel(log.DebugLevel)
		go func() {
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	}
	log.SetFormatter(formatter)
	supervisor.Run()
}
