package db

import (
	"testing"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
)

func newTestDB(t *testing.T) (*Database, *Repository) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.LocalStorageConfig{DataDir: dir}

	db, err := New(cfg, testLogger())
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, NewRepository(db)
}

func TestRepository_SaveAndGetAsset(t *testing.T) {
	_, repo := newTestDB(t)

	asset := &models.Asset{
		ID:        "asset-1",
		IP:        "192.168.1.10",
		MAC:       "aa:bb:cc:dd:ee:ff",
		Hostname:  "workstation",
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := repo.SaveAsset(asset); err != nil {
		t.Fatalf("SaveAsset failed: %v", err)
	}

	got, err := repo.GetAsset("asset-1")
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if got.IP != "192.168.1.10" {
		t.Fatalf("expected IP 192.168.1.10, got %s", got.IP)
	}
	if got.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("expected MAC aa:bb:cc:dd:ee:ff, got %s", got.MAC)
	}
}

func TestRepository_GetAssetNotFound(t *testing.T) {
	_, repo := newTestDB(t)

	_, err := repo.GetAsset("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
}

func TestRepository_ListAssets(t *testing.T) {
	_, repo := newTestDB(t)

	for i, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		repo.SaveAsset(&models.Asset{
			ID:        ip,
			IP:        ip,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			Ports:     []int{80, 443},
			Hostname:  "",
			MAC:       "",
			OS:        "",
		})
		_ = i
	}

	assets, err := repo.ListAssets()
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if len(assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(assets))
	}
}

func TestRepository_LogEvent(t *testing.T) {
	_, repo := newTestDB(t)

	ev := &models.NetworkEvent{
		ID:          "ev-1",
		Type:        models.EventDNS,
		Source:      "10.0.0.1",
		Destination: "8.8.8.8",
		Timestamp:   time.Now(),
		Metadata: map[string]any{
			"name": "example.com.",
		},
	}

	if err := repo.LogEvent(ev); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}
}

func TestRepository_UpsertAsset(t *testing.T) {
	_, repo := newTestDB(t)

	asset := &models.Asset{
		ID:       "asset-u",
		IP:       "10.0.0.1",
		Hostname: "before",
	}
	repo.SaveAsset(asset)

	// Update
	asset.Hostname = "after"
	repo.SaveAsset(asset)

	got, _ := repo.GetAsset("asset-u")
	if got.Hostname != "after" {
		t.Fatalf("expected hostname 'after', got '%s'", got.Hostname)
	}
}
