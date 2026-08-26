package main

import (
	"flag"
	"time"
)

type config struct {
	addr           string
	batchSize      int
	workers        int
	bootDelay      time.Duration
	defaultTimeout time.Duration
	handleLimit    int
	scaleInterval  time.Duration
	consolePath    string
	showVersion    bool
}

func parseConfig() config {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", "127.0.0.1:8080", "HTTP listen address")
	flag.IntVar(&cfg.batchSize, "batch-size", 8, "calls per dispatch batch")
	flag.IntVar(&cfg.workers, "workers", 3, "dispatcher worker count")
	flag.DurationVar(&cfg.bootDelay, "boot-delay", 30*time.Millisecond, "simulated cold start delay")
	flag.DurationVar(&cfg.defaultTimeout, "timeout", 5*time.Second, "default call timeout")
	flag.IntVar(&cfg.handleLimit, "handle-limit", 64, "runtime handle ceiling")
	flag.DurationVar(&cfg.scaleInterval, "scale-interval", 2*time.Second, "autoscaler interval")
	flag.StringVar(&cfg.consolePath, "console", "web/console.html", "console page path")
	flag.BoolVar(&cfg.showVersion, "version", false, "print version and exit")
	flag.Parse()
	return cfg
}
