package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// AuditEntry is an immutable, hash-chained log entry.
type AuditEntry struct {
	Sequence  uint64    `json:"seq"`
	Timestamp time.Time `json:"ts"`
	Action    string    `json:"action"`
	Subject   string    `json:"subject"`
	Details   string    `json:"details,omitempty"`
	PrevHash  string    `json:"prev_hash"`
	Hash      string    `json:"hash"`
}

// AuditLog is an append-only, hash-chained audit log.
// Each entry's hash covers its content plus the previous entry's hash,
// forming an immutable chain — any tampering breaks the chain.
type AuditLog struct {
	mu       sync.Mutex
	path     string
	file     *os.File
	lastHash string
	seq      uint64
}

// NewAuditLog opens (or creates) an append-only audit log file.
func NewAuditLog(path string) (*AuditLog, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	al := &AuditLog{
		path:     path,
		file:     f,
		lastHash: "genesis",
	}

	// Recover last hash and sequence from existing entries
	al.recoverState()

	return al, nil
}

// Append adds a new entry to the audit log.
func (al *AuditLog) Append(action, subject, details string) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.seq++
	entry := AuditEntry{
		Sequence:  al.seq,
		Timestamp: time.Now().UTC(),
		Action:    action,
		Subject:   subject,
		Details:   details,
		PrevHash:  al.lastHash,
	}

	// Hash covers all fields except Hash itself
	entry.Hash = al.computeHash(entry)
	al.lastHash = entry.Hash

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, err := al.file.Write(append(data, '\n')); err != nil {
		return err
	}

	return al.file.Sync()
}

// Close closes the audit log file.
func (al *AuditLog) Close() error {
	return al.file.Close()
}

// Verify reads the full log and checks hash chain integrity.
// Returns nil if valid, or an error describing the first broken link.
func Verify(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var entries []AuditEntry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e AuditEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return fmt.Errorf("corrupt entry: %w", err)
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		return nil
	}

	prevHash := "genesis"
	for i, e := range entries {
		if e.PrevHash != prevHash {
			return fmt.Errorf("chain broken at seq %d: expected prev_hash %s, got %s", e.Sequence, prevHash, e.PrevHash)
		}
		expected := computeEntryHash(e)
		if e.Hash != expected {
			return fmt.Errorf("tampered entry at seq %d: expected hash %s, got %s", e.Sequence, expected, e.Hash)
		}
		prevHash = e.Hash
		_ = i
	}

	return nil
}

func (al *AuditLog) computeHash(e AuditEntry) string {
	return computeEntryHash(e)
}

func computeEntryHash(e AuditEntry) string {
	payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s",
		e.Sequence,
		e.Timestamp.Format(time.RFC3339Nano),
		e.Action,
		e.Subject,
		e.Details,
		e.PrevHash,
	)
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

func (al *AuditLog) recoverState() {
	data, err := os.ReadFile(al.path)
	if err != nil {
		return
	}
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e AuditEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		al.lastHash = e.Hash
		al.seq = e.Sequence
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
