package virtual

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// LibvirtScanner discovers VMs via the virsh CLI, avoiding CGO libvirt bindings.
type LibvirtScanner struct {
	cfg      *config.VirtualConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	uri      string // libvirt connection URI
}

// NewLibvirtScanner creates a scanner that shells out to virsh.
func NewLibvirtScanner(cfg *config.VirtualConfig, logger *logging.Logger, pl *events.Pipeline) *LibvirtScanner {
	uri := "qemu:///system"
	if hv, ok := cfg.Hypervisors["libvirt"]; ok {
		if m, ok := hv.(map[string]any); ok {
			if u, ok := m["uri"].(string); ok && u != "" {
				uri = u
			}
		}
	}
	return &LibvirtScanner{
		cfg:      cfg,
		logger:   logger,
		pipeline: pl,
		uri:      uri,
	}
}

// Enabled returns true when libvirt scanning is configured on.
func (l *LibvirtScanner) Enabled() bool {
	hv, ok := l.cfg.Hypervisors["libvirt"]
	if !ok {
		return false
	}
	m, ok := hv.(map[string]any)
	if !ok {
		return false
	}
	enabled, _ := m["enabled"].(bool)
	return enabled
}

// Scan enumerates all libvirt domains and pushes each as an Asset.
func (l *LibvirtScanner) Scan(ctx context.Context) error {
	if _, err := exec.LookPath("virsh"); err != nil {
		return err
	}

	domains, err := l.listDomains(ctx)
	if err != nil {
		return err
	}

	for _, dom := range domains {
		info, err := l.domainInfo(ctx, dom.name)
		if err != nil {
			l.logger.Warnw("Failed to query domain info", "domain", dom.name, "error", err)
			continue
		}

		ip, mac := l.domainNetwork(ctx, dom.name)

		asset := models.Asset{
			ID:       "vm-" + dom.id,
			IP:       ip,
			MAC:      mac,
			Hostname: dom.name,
			OS:       "vm/libvirt",
		}

		if l.pipeline != nil {
			l.pipeline.PushAsset(asset)
		}

		meta := map[string]any{
			"domain":    dom.name,
			"state":     dom.state,
			"vcpus":     info.vcpus,
			"memory_kb": info.memoryKB,
			"vm_id":     dom.id,
		}
		if info.autostart != "" {
			meta["autostart"] = info.autostart
		}

		if l.pipeline != nil {
			l.pipeline.PushNetworkEvent(models.NetworkEvent{
				Type:        models.EventVM,
				Source:      "libvirt",
				Destination: dom.name,
				Metadata:    meta,
				Timestamp:   time.Now(),
			})
		}

		l.logger.Infow("VM Discovered",
			"id", dom.id,
			"name", dom.name,
			"state", dom.state,
			"vcpus", info.vcpus,
			"memory_kb", info.memoryKB,
			"ip", ip,
		)
	}

	return nil
}

type domain struct {
	id    string
	name  string
	state string
}

type domInfo struct {
	vcpus     string
	memoryKB  string
	autostart string
}

// listDomains parses `virsh list --all`.
func (l *LibvirtScanner) listDomains(ctx context.Context) ([]domain, error) {
	out, err := exec.CommandContext(ctx, "virsh", "-c", l.uri, "list", "--all").Output()
	if err != nil {
		return nil, err
	}

	var domains []domain
	scanner := bufio.NewScanner(bytes.NewReader(out))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		// Skip header lines (first two lines: header + separator)
		if lineNum <= 2 {
			continue
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		domains = append(domains, domain{
			id:    fields[0],
			name:  fields[1],
			state: strings.Join(fields[2:], " "),
		})
	}
	return domains, nil
}

// domainInfo parses `virsh dominfo <name>`.
func (l *LibvirtScanner) domainInfo(ctx context.Context, name string) (domInfo, error) {
	out, err := exec.CommandContext(ctx, "virsh", "-c", l.uri, "dominfo", name).Output()
	if err != nil {
		return domInfo{}, err
	}

	info := domInfo{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "CPU(s)":
			info.vcpus = val
		case "Max memory", "Used memory":
			if info.memoryKB == "" {
				info.memoryKB = val
			}
		case "Autostart":
			info.autostart = val
		}
	}
	return info, nil
}

// domainNetwork attempts to get IP/MAC via `virsh domifaddr`.
func (l *LibvirtScanner) domainNetwork(ctx context.Context, name string) (ip, mac string) {
	out, err := exec.CommandContext(ctx, "virsh", "-c", l.uri, "domifaddr", name).Output()
	if err != nil {
		return "", ""
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			mac = fields[1]
			// IP field is "addr/prefix"
			ipField := fields[3]
			if idx := strings.Index(ipField, "/"); idx > 0 {
				ip = ipField[:idx]
			} else {
				ip = ipField
			}
			break
		}
	}
	return ip, mac
}
