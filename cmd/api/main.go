// Command api serves the REST API for the shared React front end, at
// http://127.0.0.1:8000 by default (API_ADDRESS changes it).
package main

import (
	"log"
	"net/http"
	"os"

	"go.temporal.io/sdk/client"

	"github.com/temporal-sa/adk-go-wealth-management/internal/api"
	"github.com/temporal-sa/adk-go-wealth-management/internal/chaos"
	"github.com/temporal-sa/adk-go-wealth-management/internal/config"
	"github.com/temporal-sa/adk-go-wealth-management/internal/eventstream"
	"github.com/temporal-sa/adk-go-wealth-management/internal/redisx"
)

func main() {
	cfg := config.Load()
	opts, err := config.ClientOptions()
	if err != nil {
		log.Fatalln("Unable to load the Temporal client options:", err)
	}
	c, err := client.Dial(opts)
	if err != nil {
		log.Fatalln("Unable to connect to Temporal:", err)
	}
	defer c.Close()
	rdb := redisx.NewClient()
	defer rdb.Close()

	s := &api.Server{
		Temporal:  c,
		TaskQueue: cfg.TaskQueue,
		Events:    eventstream.NewRedis(rdb),
		Chaos:     chaos.NewManager(rdb),
	}
	addr := os.Getenv("API_ADDRESS")
	if addr == "" {
		addr = "127.0.0.1:8000"
	}
	log.Printf("API listening on http://%s (Temporal %s, task queue %s)", addr, opts.HostPort, cfg.TaskQueue)
	log.Fatal(http.ListenAndServe(addr, s.Handler()))
}
