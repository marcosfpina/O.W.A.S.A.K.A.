package crypto

import (
	"bytes"
	"testing"
)

func TestVault(t *testing.T) {
	// 1. Setup
	key, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	vault, err := NewVault(key)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	// 2. Encrypt
	plaintext := []byte("The payload must be secured at all costs.")
	ciphertext, err := vault.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("Ciphertext matches plaintext (encryption failed to obfuscate)")
	}

	// 3. Decrypt
	decrypted, err := vault.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// 4. Verify match
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Decrypted text does not match original.\nGot:  %s\nWant: %s", decrypted, plaintext)
	}
}

func TestIntegrityCheck(t *testing.T) {
	key, _ := GenerateRandomBytes(32)
	vault, _ := NewVault(key)

	plaintext := []byte("Integrity matters.")
	ciphertext, _ := vault.Encrypt(plaintext)

	// Tamper with the ciphertext (flip a bit in the data)
	// Skip nonce at the beginning
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err := vault.Decrypt(ciphertext)
	if err == nil {
		t.Fatal("Decryption should have failed integrity check, but succeeded")
	}
}
