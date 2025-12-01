package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	saltLen   = 16
	keyLen    = 32
	scryptN   = 1 << 15 // adjust for performance/security
	scryptR   = 8
	scryptP   = 1
	nonceSize = 12
)

// EncryptWithPassword: returns payload containing:
// 8 bytes payload length (uint64 big-endian) + salt + nonce + ciphertext
// but we usually call this and then embed the returned bytes as "payload"
func EncryptWithPassword(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// build payload: [salt][nonce][ciphertext]
	out := make([]byte, 0, saltLen+nonceSize+len(ciphertext))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// DecryptWithPassword expects a payload produced by EncryptWithPassword (salt|nonce|ciphertext)
// and returns the plaintext.
func DecryptWithPassword(payload []byte, password string) ([]byte, error) {
	if len(payload) < saltLen+nonceSize {
		return nil, errors.New("payload too short")
	}
	salt := payload[:saltLen]
	nonce := payload[saltLen : saltLen+nonceSize]
	ct := payload[saltLen+nonceSize:]

	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

// helper to prefix a length header (uint64 BE) to payload (used by steg package)
func PrefixLength(payload []byte) []byte {
	lb := make([]byte, 8)
	binary.BigEndian.PutUint64(lb, uint64(len(payload)))
	return append(lb, payload...)
}

// helper to read length prefix (uint64 BE)
func ReadLengthPrefixed(data []byte) (uint64, []byte, bool) {
	if len(data) < 8 {
		return 0, nil, false
	}
	n := binary.BigEndian.Uint64(data[:8])
	return n, data[8:], true
}
