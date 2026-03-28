package attack_surface

import (
	"context"
	"testing"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
)

func TestPortScanner_ProbeLocalhost(t *testing.T) {
	cfg := &config.AttackSurfaceConfig{
		Enabled:         true,
		ConcurrentScans: 10,
		PortRange:       config.PortRange{Start: 1, End: 100},
		BannerGrabbing:  false,
	}
	scanner := NewPortScanner(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := scanner.ScanHost(ctx, "127.0.0.1")

	// Drain results — we don't know what ports are open, just ensure no panic
	count := 0
	for range results {
		count++
	}
	// At minimum, the scan should complete without error
	t.Logf("found %d open ports in range 1-100 on localhost", count)
}

func TestPortScanner_CancelledContext(t *testing.T) {
	cfg := &config.AttackSurfaceConfig{
		Enabled:         true,
		ConcurrentScans: 5,
		PortRange:       config.PortRange{Start: 1, End: 65535},
		BannerGrabbing:  false,
	}
	scanner := NewPortScanner(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	results := scanner.ScanHost(ctx, "127.0.0.1")

	// Should complete quickly without hanging
	count := 0
	for range results {
		count++
	}
	// With cancelled context, should get few or no results
	t.Logf("got %d results after immediate cancel", count)
}

type mockAssetLister struct {
	assets []models.Asset
}

func (m *mockAssetLister) ListAssets() ([]models.Asset, error) {
	return m.assets, nil
}

func TestService_ResolveTargetsFromAssets(t *testing.T) {
	cfg := &config.AttackSurfaceConfig{
		Enabled:         true,
		ConcurrentScans: 5,
		PortRange:       config.PortRange{Start: 80, End: 80},
	}

	lister := &mockAssetLister{
		assets: []models.Asset{
			{IP: "10.0.0.1"},
			{IP: "10.0.0.2"},
			{IP: "10.0.0.1"}, // duplicate
		},
	}

	svc := NewService(cfg, nil, nil, lister)
	targets := svc.resolveTargets()

	if len(targets) != 2 {
		t.Fatalf("expected 2 unique targets, got %d: %v", len(targets), targets)
	}
}

func TestService_FallbackToLocalhost(t *testing.T) {
	cfg := &config.AttackSurfaceConfig{
		Enabled: true,
	}

	svc := NewService(cfg, nil, nil, nil)
	targets := svc.resolveTargets()

	if len(targets) != 1 || targets[0] != "127.0.0.1" {
		t.Fatalf("expected [127.0.0.1] fallback, got %v", targets)
	}
}
