package discovery

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/google/uuid"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/metrics"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

// handleARP processes ARP packets to discover devices
func (s *Scanner) handleARP(arp *layers.ARP) {
	// We are interested in ARP replies or requests that show a device exists
	// Source IP and Source MAC are what we care about

	srcIP := net.IP(arp.SourceProtAddress)
	srcMAC := net.HardwareAddr(arp.SourceHwAddress)

	// Log discovery (in a real implementation, this would update an Asset Store)
	// We avoid logging our own traffic if possible, but for now simple logging is enough
	// Differentiate between Request and Reply for context
	op := "Unknown"
	switch arp.Operation {
	case layers.ARPRequest:
		op = "Request"
	case layers.ARPReply:
		op = "Reply"
	}

	// Log every unique discovery? No, too noisy.
	// For MVP, just log it.
	s.logger.Infow("Device Detected (ARP)",
		"ip", srcIP.String(),
		"mac", srcMAC.String(),
		"operation", op,
	)

	metrics.AssetsDiscovered.Inc()

	if s.pipeline != nil {
		evt := models.NetworkEvent{
			ID:          uuid.NewString(),
			Type:        models.EventARP,
			Timestamp:   time.Now().UTC(),
			Source:      srcIP.String(),
			Destination: "SIEM_DISCOVERY",
			Metadata: map[string]any{
				"ip":        srcIP.String(),
				"mac":       srcMAC.String(),
				"operation": op,
			},
		}
		s.pipeline.PushNetworkEvent(evt)
	}
}

func ipToInt(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}
