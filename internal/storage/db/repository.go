package db

import (
	"encoding/json"
	"fmt"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	bolt "go.etcd.io/bbolt"
)

// Repository manages structural interaction with BoltDB schema
type Repository struct {
	db *Database
}

// NewRepository creates a new DAO
func NewRepository(db *Database) *Repository {
	return &Repository{db: db}
}

// SaveAsset upserts network/physical assets by string ID
func (r *Repository) SaveAsset(a *models.Asset) error {
	return r.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketAssets))
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		return b.Put([]byte(a.ID), data)
	})
}

// GetAsset reads a single asset resolving it off json
func (r *Repository) GetAsset(id string) (*models.Asset, error) {
	var a models.Asset
	err := r.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketAssets))
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("asset not found")
		}
		return json.Unmarshal(data, &a)
	})
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// LogEvent streams a forensic event to BoltDB mapping an ID to structure
func (r *Repository) LogEvent(e *models.NetworkEvent) error {
	return r.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketEvents))
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return b.Put([]byte(e.ID), data)
	})
}
