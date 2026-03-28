package attack_surface

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// PortResult holds the result of a single port scan
type PortResult struct {
	Host   string
	Port   int
	Open   bool
	Banner string
}

// PortScanner performs concurrent TCP port scanning
type PortScanner struct {
	cfg    *config.AttackSurfaceConfig
	logger *logging.Logger
}

// NewPortScanner creates a new port scanner
func NewPortScanner(cfg *config.AttackSurfaceConfig, logger *logging.Logger) *PortScanner {
	return &PortScanner{cfg: cfg, logger: logger}
}

// ScanHost scans all ports in the configured range for a given host
func (s *PortScanner) ScanHost(ctx context.Context, host string) <-chan PortResult {
	results := make(chan PortResult, 256)

	go func() {
		defer close(results)

		portCh := make(chan int, s.cfg.ConcurrentScans)
		var wg sync.WaitGroup

		workers := s.cfg.ConcurrentScans
		if workers <= 0 {
			workers = 100
		}

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for port := range portCh {
					if ctx.Err() != nil {
						return
					}
					open, banner := s.probe(host, port)
					if open {
						results <- PortResult{Host: host, Port: port, Open: true, Banner: banner}
					}
				}
			}()
		}

		start := s.cfg.PortRange.Start
		end := s.cfg.PortRange.End
		if start <= 0 {
			start = 1
		}
		if end <= 0 || end > 65535 {
			end = 65535
		}

		for p := start; p <= end; p++ {
			select {
			case <-ctx.Done():
				close(portCh)
				wg.Wait()
				return
			case portCh <- p:
			}
		}
		close(portCh)
		wg.Wait()
	}()

	return results
}

// probe attempts a TCP connection and optionally grabs a banner
func (s *PortScanner) probe(host string, port int) (bool, string) {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return false, ""
	}
	defer conn.Close()

	banner := ""
	if s.cfg.BannerGrabbing {
		_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		if n > 0 {
			banner = string(buf[:n])
		}
	}

	return true, banner
}
