package virtual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// DockerScanner retrieves container metadata purely over Unix Domain Sockets 
// bypassing massive third-party package dependency chains.
type DockerScanner struct {
	cfg      *config.ContainerConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	client   *http.Client
}

// NewDockerScanner configures the socket-level HTTP client
func NewDockerScanner(cfg *config.ContainerConfig, logger *logging.Logger, pl *events.Pipeline) *DockerScanner {
	socketPath := cfg.DockerSocket
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	return &DockerScanner{
		cfg:      cfg,
		logger:   logger,
		pipeline: pl,
		client:   client,
	}
}

// Scan touches the Docker Engine `/containers/json` API
func (d *DockerScanner) Scan(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://unix/v1.41/containers/json", nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("socket reachability failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker API returned status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var containers []struct {
		Id      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		State   string   `json:"State"`
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress   string `json:"IPAddress"`
				MacAddress  string `json:"MacAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}

	if err := json.Unmarshal(data, &containers); err != nil {
		return err
	}

	for _, c := range containers {
		name := "unknown"
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		
		ip := ""
		mac := ""
		for _, netw := range c.NetworkSettings.Networks {
			ip = netw.IPAddress
			mac = netw.MacAddress
			break // Record the primary network configuration
		}

		shortID := c.Id
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		if d.pipeline != nil {
			d.pipeline.PushAsset(models.Asset{
				ID:       "docker-" + shortID,
				IP:       ip,
				MAC:      mac,
				Hostname: name,
				OS:       "container/" + c.Image,
			})
		}
		
		d.logger.Infow("Virtual Container Discovered",
			"id", shortID,
			"name", name,
			"image", c.Image,
			"ip", ip,
		)
	}

	return nil
}
