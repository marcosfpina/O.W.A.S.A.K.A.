package virtual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// ResourceTracker collects CPU/memory/IO stats for running Docker containers.
type ResourceTracker struct {
	client   *http.Client
	logger   *logging.Logger
	pipeline *events.Pipeline
}

// NewResourceTracker reuses the same Unix-socket HTTP client as DockerScanner.
func NewResourceTracker(client *http.Client, logger *logging.Logger, pl *events.Pipeline) *ResourceTracker {
	return &ResourceTracker{
		client:   client,
		logger:   logger,
		pipeline: pl,
	}
}

// dockerStats is the subset of Docker /stats we care about.
type dockerStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PrecpuStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IOServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}

// Collect gathers resource stats for a single container and emits a VM event.
func (rt *ResourceTracker) Collect(ctx context.Context, containerID, containerName string) error {
	url := fmt.Sprintf("http://unix/v1.41/containers/%s/stats?stream=false", containerID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := rt.client.Do(req)
	if err != nil {
		return fmt.Errorf("stats request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker stats API returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var stats dockerStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return err
	}

	cpuPct := calculateCPUPercent(stats)
	memPct := 0.0
	if stats.MemoryStats.Limit > 0 {
		memPct = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100
	}

	var rxBytes, txBytes uint64
	for _, netStats := range stats.Networks {
		rxBytes += netStats.RxBytes
		txBytes += netStats.TxBytes
	}

	var blkRead, blkWrite uint64
	for _, entry := range stats.BlkioStats.IOServiceBytesRecursive {
		switch entry.Op {
		case "read", "Read":
			blkRead += entry.Value
		case "write", "Write":
			blkWrite += entry.Value
		}
	}

	shortID := containerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	meta := map[string]any{
		"container_id":   shortID,
		"container_name": containerName,
		"cpu_percent":    fmt.Sprintf("%.2f", cpuPct),
		"memory_usage":   stats.MemoryStats.Usage,
		"memory_limit":   stats.MemoryStats.Limit,
		"memory_percent": fmt.Sprintf("%.2f", memPct),
		"net_rx_bytes":   rxBytes,
		"net_tx_bytes":   txBytes,
		"blk_read":       blkRead,
		"blk_write":      blkWrite,
	}

	if rt.pipeline != nil {
		rt.pipeline.PushNetworkEvent(models.NetworkEvent{
			Type:        models.EventVM,
			Source:      "docker-" + shortID,
			Destination: "resource-stats",
			Metadata:    meta,
			Timestamp:   time.Now(),
		})
	}

	return nil
}

// CollectAll fetches the container list and collects stats for each running container.
func (rt *ResourceTracker) CollectAll(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://unix/v1.41/containers/json", nil)
	if err != nil {
		return err
	}

	resp, err := rt.client.Do(req)
	if err != nil {
		return fmt.Errorf("container list failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker API returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var containers []struct {
		Id    string   `json:"Id"`
		Names []string `json:"Names"`
		State string   `json:"State"`
	}
	if err := json.Unmarshal(data, &containers); err != nil {
		return err
	}

	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		name := "unknown"
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		if err := rt.Collect(ctx, c.Id, name); err != nil {
			rt.logger.Warnw("Stats collection failed", "container", name, "error", err)
		}
	}

	return nil
}

// calculateCPUPercent computes CPU usage percentage from Docker stats delta.
func calculateCPUPercent(s dockerStats) float64 {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage - s.PrecpuStats.CPUUsage.TotalUsage)
	systemDelta := float64(s.CPUStats.SystemCPUUsage - s.PrecpuStats.SystemCPUUsage)
	if systemDelta <= 0 || cpuDelta <= 0 {
		return 0
	}
	cpus := s.CPUStats.OnlineCPUs
	if cpus == 0 {
		cpus = 1
	}
	return (cpuDelta / systemDelta) * float64(cpus) * 100
}
