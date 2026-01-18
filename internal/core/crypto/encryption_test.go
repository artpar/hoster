package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Data
// =============================================================================

// Sample ed25519 private key for testing (DO NOT USE IN PRODUCTION)
const testSSHPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACATvKRI3HN94cf22YT2iPCrGpv/6QSBhognjx/zTPE50wAAAJgmOTMMJjkz
DAAAAAtzc2gtZWQyNTUxOQAAACATvKRI3HN94cf22YT2iPCrGpv/6QSBhognjx/zTPE50w
AAAEBCkOPNNcK4D15gcc5fbSCMAcbHJ0XjxXf9R+HS16TUpxO8pEjcc33hx/bZhPaI8Ksa
m//pBIGGiCePH/NM8TnTAAAAEHRlc3RAZXhhbXBsZS5jb20BAgMEBQ==
-----END OPENSSH PRIVATE KEY-----`

// =============================================================================
// DeriveKey Tests
// =============================================================================

func TestDeriveKey(t *testing.T) {
	key := DeriveKey("my-secret-passphrase")
	assert.Len(t, key, 32) // SHA-256 produces 32 bytes
}

func TestDeriveKey_Deterministic(t *testing.T) {
	key1 := DeriveKey("same-passphrase")
	key2 := DeriveKey("same-passphrase")
	assert.Equal(t, key1, key2)
}

func TestDeriveKey_DifferentInput(t *testing.T) {
	key1 := DeriveKey("passphrase1")
	key2 := DeriveKey("passphrase2")
	assert.NotEqual(t, key1, key2)
}

// =============================================================================
// Encrypt/Decrypt Tests
// =============================================================================

func TestEncrypt_Decrypt_Roundtrip(t *testing.T) {
	plaintext := []byte("This is a secret message!")
	key := DeriveKey("test-encryption-key")

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	plaintext := []byte("Same message")
	key := DeriveKey("test-key")

	ciphertext1, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	ciphertext2, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Same plaintext should produce different ciphertext (different nonces)
	assert.NotEqual(t, ciphertext1, ciphertext2)
}

func TestEncrypt_KeyTooShort(t *testing.T) {
	plaintext := []byte("test")
	shortKey := []byte("too-short") // Less than 32 bytes

	_, err := Encrypt(plaintext, shortKey)
	assert.ErrorIs(t, err, ErrKeyTooShort)
}

func TestDecrypt_KeyTooShort(t *testing.T) {
	ciphertext := []byte("some-ciphertext-data-that-is-long-enough")
	shortKey := []byte("too-short")

	_, err := Decrypt(ciphertext, shortKey)
	assert.ErrorIs(t, err, ErrKeyTooShort)
}

func TestDecrypt_WrongKey(t *testing.T) {
	plaintext := []byte("secret")
	key1 := DeriveKey("correct-key")
	key2 := DeriveKey("wrong-key")

	ciphertext, err := Encrypt(plaintext, key1)
	require.NoError(t, err)

	_, err = Decrypt(ciphertext, key2)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	key := DeriveKey("test-key")
	shortCiphertext := []byte("short") // Too short to contain nonce

	_, err := Decrypt(shortCiphertext, key)
	assert.ErrorIs(t, err, ErrInvalidCiphertext)
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	plaintext := []byte("secret")
	key := DeriveKey("test-key")

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Corrupt the ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = Decrypt(ciphertext, key)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	plaintext := []byte{}
	key := DeriveKey("test-key")

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext) // Contains nonce + auth tag

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestEncrypt_LargePlaintext(t *testing.T) {
	// 1 MB of data
	plaintext := bytes.Repeat([]byte("x"), 1024*1024)
	key := DeriveKey("test-key")

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// =============================================================================
// Base64 Encoding Tests
// =============================================================================

func TestEncryptToBase64_DecryptFromBase64(t *testing.T) {
	plaintext := []byte("secret data")
	key := DeriveKey("test-key")

	encoded, err := EncryptToBase64(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decrypted, err := DecryptFromBase64(encoded, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecryptFromBase64_InvalidBase64(t *testing.T) {
	key := DeriveKey("test-key")

	_, err := DecryptFromBase64("not-valid-base64!@#", key)
	assert.Error(t, err)
}

// =============================================================================
// SSH Key Tests
// =============================================================================

func TestEncryptSSHKey_DecryptSSHKey(t *testing.T) {
	privateKey := []byte(testSSHPrivateKey)
	encryptionKey := DeriveKey("platform-master-secret")

	encrypted, err := EncryptSSHKey(privateKey, encryptionKey)
	require.NoError(t, err)
	assert.NotEqual(t, privateKey, encrypted)

	decrypted, err := DecryptSSHKey(encrypted, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, privateKey, decrypted)
}

func TestValidateSSHPrivateKey_Valid(t *testing.T) {
	err := ValidateSSHPrivateKey([]byte(testSSHPrivateKey))
	assert.NoError(t, err)
}

func TestValidateSSHPrivateKey_Invalid(t *testing.T) {
	invalidKey := []byte("not a valid ssh key")
	err := ValidateSSHPrivateKey(invalidKey)
	assert.ErrorIs(t, err, ErrInvalidSSHKey)
}

func TestValidateSSHPrivateKey_Empty(t *testing.T) {
	err := ValidateSSHPrivateKey([]byte{})
	assert.ErrorIs(t, err, ErrInvalidSSHKey)
}

func TestParseSSHPrivateKey_Valid(t *testing.T) {
	signer, err := ParseSSHPrivateKey([]byte(testSSHPrivateKey))
	require.NoError(t, err)
	assert.NotNil(t, signer)
}

func TestParseSSHPrivateKey_Invalid(t *testing.T) {
	_, err := ParseSSHPrivateKey([]byte("invalid"))
	assert.ErrorIs(t, err, ErrInvalidSSHKey)
}

func TestGetSSHPublicKeyFingerprint(t *testing.T) {
	fingerprint, err := GetSSHPublicKeyFingerprint([]byte(testSSHPrivateKey))
	require.NoError(t, err)
	assert.True(t, len(fingerprint) > 0)
	assert.Contains(t, fingerprint, "SHA256:")
}

func TestGetSSHPublicKeyFingerprint_InvalidKey(t *testing.T) {
	_, err := GetSSHPublicKeyFingerprint([]byte("invalid"))
	assert.ErrorIs(t, err, ErrInvalidSSHKey)
}

func TestGetSSHPublicKey(t *testing.T) {
	pubKey, err := GetSSHPublicKey([]byte(testSSHPrivateKey))
	require.NoError(t, err)
	assert.Contains(t, pubKey, "ssh-ed25519")
}

func TestGetSSHPublicKey_InvalidKey(t *testing.T) {
	_, err := GetSSHPublicKey([]byte("invalid"))
	assert.ErrorIs(t, err, ErrInvalidSSHKey)
}

// =============================================================================
// Key Length Edge Cases
// =============================================================================

func TestEncrypt_ExactlyKey32Bytes(t *testing.T) {
	plaintext := []byte("test")
	key := make([]byte, 32)
	copy(key, []byte("exactly-32-bytes-key-0123456789"))

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_LongerKey(t *testing.T) {
	plaintext := []byte("test")
	key := make([]byte, 64) // Longer than 32 bytes
	copy(key, []byte("this-is-a-much-longer-key-that-exceeds-32-bytes-limit"))

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
