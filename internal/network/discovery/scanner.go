package discovery

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Scanner handles passive network discovery
type Scanner struct {
	cfg       *config.ScanConfig
	logger    *logging.Logger
	handle    *pcap.Handle
	stopChan  chan struct{}
	isRunning bool
}

// NewScanner creates a new passive scanner
func NewScanner(cfg *config.ScanConfig, logger *logging.Logger) *Scanner {
	return &Scanner{
		cfg:      cfg,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start begins capturing and analyzing packets
func (s *Scanner) Start(interfaceName string) error {
	if s.isRunning {
		return nil
	}

	var err error
	// Open device for capturing
	// Promiscuous mode = true, SnapLen = 1024 (enough for headers)
	s.handle, err = pcap.OpenLive(interfaceName, 1024, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %w", interfaceName, err)
	}

	// Set filter to capture ARP
	// We can expand this later to capture mDNS (udp port 5353)
	if err := s.handle.SetBPFFilter("arp"); err != nil {
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	s.isRunning = true
	s.logger.Infow("Passive Scanner started", "interface", interfaceName)

	go s.captureLoop()

	return nil
}

// Stop stops the packet capture
func (s *Scanner) Stop() {
	if !s.isRunning {
		return
	}
	close(s.stopChan)
	if s.handle != nil {
		s.handle.Close()
	}
	s.isRunning = false
	s.logger.Info("Passive Scanner stopped")
}

func (s *Scanner) captureLoop() {
	packetSource := gopacket.NewPacketSource(s.handle, s.handle.LinkType())
	packets := packetSource.Packets()

	for {
		select {
		case <-s.stopChan:
			return
		case packet, ok := <-packets:
			if !ok {
				return
			}
			s.processPacket(packet)
		}
	}
}

func (s *Scanner) processPacket(packet gopacket.Packet) {
	// Check for ARP layer
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		s.handleARP(arp)
	}
}
