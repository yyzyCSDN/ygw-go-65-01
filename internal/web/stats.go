package web

import (
	"fnexec/internal/cold"
	"fnexec/internal/exec"
	"fnexec/internal/func"
	"fnexec/internal/queue"
	"fnexec/internal/scale"
	"fnexec/internal/trigger"
)

// Stats is the serializable snapshot served by /api/stats.
type Stats struct {
	Version       string               `json:"version"`
	UptimeSeconds int64                `json:"uptime_seconds"`
	Functions     []funcs.Entry        `json:"functions"`
	Queue         queue.Snapshot       `json:"queue"`
	Instances     []scale.InstanceView `json:"instances"`
	Routes        map[string][]string  `json:"routes"`
	Exec          exec.StatsSnapshot   `json:"exec"`
	Retries       int64                `json:"retries"`
	ColdCache     int                  `json:"cold_cache"`
	ColdInstances []cold.InstanceView  `json:"cold_instances"`
	Trigger       trigger.Stats        `json:"trigger"`
}
