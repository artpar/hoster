// Package crypto provides encryption utilities for sensitive data like SSH keys.
// This is part of the Functional Core - all functions are pure with no I/O.
//
// SSH private keys are encrypted at rest using AES-256-GCM.
// The encryption key should be derived from a platform master secret.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// =============================================================================
// Errors
// =============================================================================

var (
	// ErrKeyTooShort is returned when the encryption key is too short.
	ErrKeyTooShort = errors.New("encryption key must be at least 32 bytes")

	// ErrInvalidCiphertext is returned when decryption fails due to invalid ciphertext.
	ErrInvalidCiphertext = errors.New("invalid ciphertext: too short")

	// ErrDecryptionFailed is returned when decryption fails (wrong key or corrupted data).
	ErrDecryptionFailed = errors.New("decryption failed: authentication tag mismatch")

	// ErrInvalidSSHKey is returned when the SSH key cannot be parsed.
	ErrInvalidSSHKey = errors.New("invalid SSH private key format")
)

// =============================================================================
// Key Derivation
// =============================================================================

// DeriveKey derives a 32-byte AES-256 key from a passphrase using SHA-256.
// This is a simple key derivation function. For production use, consider
// using a proper KDF like Argon2, scrypt, or PBKDF2.
//
// Note: This function is deterministic - same input always produces same output.
func DeriveKey(passphrase string) []byte {
	hash := sha256.Sum256([]byte(passphrase))
	return hash[:]
}

// =============================================================================
// AES-256-GCM Encryption
// =============================================================================

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// The key must be exactly 32 bytes (256 bits).
//
// The ciphertext format is: nonce (12 bytes) || encrypted data || auth tag (16 bytes)
//
// Returns encrypted bytes or error if encryption fails.
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) < 32 {
		return nil, ErrKeyTooShort
	}

	// Use exactly 32 bytes for AES-256
	key = key[:32]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// The key must be exactly 32 bytes (256 bits).
//
// Returns decrypted plaintext or error if decryption fails.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) < 32 {
		return nil, ErrKeyTooShort
	}

	// Use exactly 32 bytes for AES-256
	key = key[:32]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// =============================================================================
// Base64 Encoding Variants
// =============================================================================

// EncryptToBase64 encrypts plaintext and returns base64-encoded ciphertext.
// Useful for storing encrypted data in text fields (JSON, environment variables).
func EncryptToBase64(plaintext, key []byte) (string, error) {
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 decrypts base64-encoded ciphertext.
func DecryptFromBase64(encoded string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, key)
}

// =============================================================================
// SSH Key Utilities
// =============================================================================

// EncryptSSHKey encrypts an SSH private key with the provided encryption key.
// Returns the encrypted key bytes suitable for database storage.
func EncryptSSHKey(privateKey, encryptionKey []byte) ([]byte, error) {
	return Encrypt(privateKey, encryptionKey)
}

// DecryptSSHKey decrypts an SSH private key that was encrypted with EncryptSSHKey.
func DecryptSSHKey(encryptedKey, encryptionKey []byte) ([]byte, error) {
	return Decrypt(encryptedKey, encryptionKey)
}

// ValidateSSHPrivateKey validates that the given bytes are a valid SSH private key.
// Returns nil if valid, error otherwise.
func ValidateSSHPrivateKey(privateKey []byte) error {
	_, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return ErrInvalidSSHKey
	}
	return nil
}

// ParseSSHPrivateKey parses an SSH private key and returns the signer.
func ParseSSHPrivateKey(privateKey []byte) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, ErrInvalidSSHKey
	}
	return signer, nil
}

// GetSSHPublicKeyFingerprint returns the SHA256 fingerprint of the public key
// derived from the private key.
func GetSSHPublicKeyFingerprint(privateKey []byte) (string, error) {
	signer, err := ParseSSHPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	// Get the public key from the signer
	pubKey := signer.PublicKey()

	// Calculate SHA256 fingerprint
	hash := sha256.Sum256(pubKey.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:]), nil
}

// GenerateSSHKeyPair generates a new Ed25519 SSH key pair.
// Returns the private key in PEM format and the public key in OpenSSH authorized_keys format.
func GenerateSSHKeyPair() (privateKeyPEM []byte, publicKey string, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	sshPrivKey, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return nil, "", fmt.Errorf("marshal private key: %w", err)
	}

	pemBytes := pem.EncodeToMemory(sshPrivKey)

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, "", fmt.Errorf("create public key: %w", err)
	}

	authorizedKey := string(ssh.MarshalAuthorizedKey(sshPubKey))
	return pemBytes, authorizedKey, nil
}

// GetSSHPublicKey returns the OpenSSH authorized_keys format public key
// derived from the private key.
func GetSSHPublicKey(privateKey []byte) (string, error) {
	signer, err := ParseSSHPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	pubKey := signer.PublicKey()
	return string(ssh.MarshalAuthorizedKey(pubKey)), nil
}
