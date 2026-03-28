package nas

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Mounter handles NFS/SMB mount operations using system mount commands.
type Mounter struct {
	cfg    *config.NASConfig
	logger *logging.Logger
}

// NewMounter creates a NAS mounter.
func NewMounter(cfg *config.NASConfig, logger *logging.Logger) *Mounter {
	return &Mounter{cfg: cfg, logger: logger}
}

// Mount mounts the NAS share to the configured mount point.
func (m *Mounter) Mount(ctx context.Context) error {
	if err := os.MkdirAll(m.cfg.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", m.cfg.MountPoint, err)
	}

	// Check if already mounted
	if m.IsMounted() {
		m.logger.Infow("NAS already mounted", "mount", m.cfg.MountPoint)
		return nil
	}

	timeout := time.Duration(m.cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch strings.ToLower(m.cfg.Type) {
	case "nfs":
		cmd = m.nfsMount(cmdCtx)
	case "smb", "cifs":
		cmd = m.smbMount(cmdCtx)
	default:
		return fmt.Errorf("unsupported NAS type: %s (use 'nfs' or 'smb')", m.cfg.Type)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %w — output: %s", err, strings.TrimSpace(string(out)))
	}

	m.logger.Infow("NAS mounted", "type", m.cfg.Type, "host", m.cfg.Host, "share", m.cfg.Share, "mount", m.cfg.MountPoint)
	return nil
}

// Unmount unmounts the NAS share.
func (m *Mounter) Unmount(ctx context.Context) error {
	if !m.IsMounted() {
		return nil
	}
	cmd := exec.CommandContext(ctx, "umount", m.cfg.MountPoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unmount failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	m.logger.Infow("NAS unmounted", "mount", m.cfg.MountPoint)
	return nil
}

// IsMounted checks if the mount point is active by reading /proc/mounts.
func (m *Mounter) IsMounted() bool {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), m.cfg.MountPoint)
}

func (m *Mounter) nfsMount(ctx context.Context) *exec.Cmd {
	source := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Share)
	opts := "noatime,soft,timeo=100,retrans=3"
	return exec.CommandContext(ctx, "mount", "-t", "nfs", "-o", opts, source, m.cfg.MountPoint)
}

func (m *Mounter) smbMount(ctx context.Context) *exec.Cmd {
	source := fmt.Sprintf("//%s/%s", m.cfg.Host, m.cfg.Share)
	opts := fmt.Sprintf("username=%s,password=%s,iocharset=utf8,vers=3.0",
		m.cfg.Username, m.cfg.Password)
	return exec.CommandContext(ctx, "mount", "-t", "cifs", "-o", opts, source, m.cfg.MountPoint)
}
