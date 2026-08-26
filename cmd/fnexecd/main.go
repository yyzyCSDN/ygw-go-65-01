package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fnexec/internal/engine"
)

func main() {
	cfg := parseConfig()
	if cfg.showVersion {
		fmt.Println("fnexecd " + engine.Version)
		return
	}
	consoleHTML, err := os.ReadFile(cfg.consolePath)
	if err != nil {
		log.Fatalf("read console page: %v", err)
	}
	eng, err := engine.New(engine.Options{
		HTTPAddr:       cfg.addr,
		BatchSize:      cfg.batchSize,
		Workers:        cfg.workers,
		BootDelay:      cfg.bootDelay,
		DefaultTimeout: cfg.defaultTimeout,
		HandleLimit:    cfg.handleLimit,
		ScaleInterval:  cfg.scaleInterval,
		ConsoleHTML:    consoleHTML,
	})
	if err != nil {
		log.Fatalf("build engine: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := eng.Start(ctx); err != nil {
		log.Fatalf("start engine: %v", err)
	}
	log.Printf("fnexecd %s listening on %s", engine.Version, eng.Addr())
	probeReady(eng.Addr())
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := eng.Stop(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("fnexecd stopped")
}

func probeReady(addr string) {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for attempt := 0; attempt < 20; attempt++ {
		resp, err := client.Get("http://" + addr + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("startup probe: healthz ok")
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	log.Printf("startup probe: healthz not reachable within budget")
}
