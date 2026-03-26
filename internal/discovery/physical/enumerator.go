package physical

import (
	"context"
	"os"
	"path/filepath"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Device represents a discovered physical hardware device
type Device struct {
	Bus     string
	ID      string
	Vendor  string
	Product string
	Class   string
}

// Enumerator discovers physical devices via the Linux sysfs interface
type Enumerator struct {
	cfg      *config.PhysicalConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
}

// NewEnumerator creates a new hardware enumerator
func NewEnumerator(cfg *config.PhysicalConfig, logger *logging.Logger, pl *events.Pipeline) *Enumerator {
	return &Enumerator{cfg: cfg, logger: logger, pipeline: pl}
}

// Enumerate reads all devices from the configured buses
func (e *Enumerator) Enumerate(_ context.Context) {
	buses := e.cfg.Devices
	if len(buses) == 0 {
		buses = []string{"usb", "pci"}
	}

	for _, bus := range buses {
		sysPath := filepath.Join("/sys/bus", bus, "devices")
		entries, err := os.ReadDir(sysPath)
		if err != nil {
			e.logger.Warnw("Cannot enumerate bus",
				"bus", bus, "path", sysPath, "error", err)
			continue
		}

		for _, entry := range entries {
			devicePath := filepath.Join(sysPath, entry.Name())
			dev := Device{
				Bus:     bus,
				ID:      entry.Name(),
				Vendor:  readSysAttr(devicePath, "idVendor", "vendor"),
				Product: readSysAttr(devicePath, "idProduct", "device"),
				Class:   readSysAttr(devicePath, "bDeviceClass", "class"),
			}
			
			if e.pipeline != nil {
				e.pipeline.PushAsset(models.Asset{
					ID:       dev.Bus + "-" + dev.ID,
					Hostname: dev.Vendor + " / " + dev.Product,
					MAC:      "system-hardware",
					OS:       "linux-sysfs",
				})
			} else {
				e.logger.Infow("Physical Device Detected",
					"bus", dev.Bus,
					"id", dev.ID,
					"vendor", dev.Vendor,
					"product", dev.Product,
					"class", dev.Class,
				)
			}
		}
	}
}

// readSysAttr reads a sysfs attribute, trying multiple filenames
func readSysAttr(devicePath string, names ...string) string {
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(devicePath, name))
		if err == nil {
			return string(data)
		}
	}
	return ""
}
