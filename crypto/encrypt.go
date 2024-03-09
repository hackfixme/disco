package crypto

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	// Maximum size of each encrypted chunk of data. NaCl is recommended for
	// encrypting "small" messages, so large data is split into 16KB chunks.
	chunkSize = 16 * 1024 // 16KB
	nonceSize = 24
)

// EncryptSym performs symmetric encryption of the plaintext data using NaCl
// primitives (Curve25519, XSalsa20 and Poly1305).
func EncryptSym(plaintext io.Reader, secretKey *[32]byte) (io.Reader, error) {
	return encrypt(plaintext, nil, secretKey)
}

// EncryptAsym performs asymmetric encryption of the plaintext data using NaCl
// primitives (Curve25519, XSalsa20 and Poly1305).
func EncryptAsym(plaintext io.Reader, publicKey, privateKey *[32]byte) (io.Reader, error) {
	return encrypt(plaintext, publicKey, privateKey)
}

func encrypt(in io.Reader, publicKey, privateKey *[32]byte) (io.Reader, error) {
	var encrypt func(out, message []byte, nonce *[nonceSize]byte) []byte
	if publicKey == nil {
		encrypt = func(out, message []byte, nonce *[nonceSize]byte) []byte {
			return secretbox.Seal(out, message, nonce, privateKey)
		}
	} else {
		encrypt = func(out, message []byte, nonce *[nonceSize]byte) []byte {
			return box.Seal(out, message, nonce, publicKey, privateKey)
		}
	}

	var (
		buf = make([]byte, chunkSize)
		out = &bytes.Buffer{}
	)

	for {
		n, err := in.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		nonce, err := generateNonce()
		if err != nil {
			return nil, fmt.Errorf("failed generating nonce: %w", err)
		}

		encrypted := encrypt(nonce[:], buf[:n], nonce)

		_, err = out.Write(encrypted)
		if err != nil {
			return nil, fmt.Errorf("failed writing encrypted data: %w", err)
		}
	}

	return out, nil
}

// DecryptSym performs symmetric decryption of the in ciphertext data using NaCl
// primitives (Curve25519, XSalsa20 and Poly1305).
func DecryptSym(ciphertext io.Reader, secretKey *[32]byte) (io.Reader, error) {
	return decrypt(ciphertext, nil, secretKey)
}

// DecryptAsym performs asymmetric decryption of the ciphertext data using NaCl
// primitives (Curve25519, XSalsa20 and Poly1305).
func DecryptAsym(ciphertext io.Reader, publicKey, privateKey *[32]byte) (io.Reader, error) {
	return decrypt(ciphertext, publicKey, privateKey)
}

func decrypt(in io.Reader, publicKey, privateKey *[32]byte) (io.Reader, error) {
	var decrypt func(out, data []byte, nonce *[nonceSize]byte) ([]byte, bool)
	if publicKey == nil {
		decrypt = func(out, data []byte, nonce *[nonceSize]byte) ([]byte, bool) {
			return secretbox.Open(out, data, nonce, privateKey)
		}
	} else {
		decrypt = func(out, data []byte, nonce *[nonceSize]byte) ([]byte, bool) {
			return box.Open(out, data, nonce, publicKey, privateKey)
		}
	}

	var (
		buf = make([]byte, nonceSize+chunkSize+box.Overhead)
		out = &bytes.Buffer{}
	)

	for {
		// Read the payload
		n, err := in.Read(buf)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed reading payload from buffer: %w", err)
		}
		if n == 0 {
			break
		}

		// Unpack the payload
		var nonce [nonceSize]byte
		copy(nonce[:], buf[:nonceSize])
		encrypted := buf[nonceSize:n]

		decrypted, ok := decrypt(nil, encrypted, &nonce)
		if !ok {
			return nil, errors.New("failed decrypting chunk")
		}

		_, err = out.Write(decrypted)
		if err != nil {
			return nil, fmt.Errorf("failed writing decrypted data: %w", err)
		}
	}

	return out, nil
}

// DecodeKey decodes and validates an encryption key.
func DecodeKey(keyEnc string) (*[32]byte, error) {
	keyDec, err := base58.Decode(keyEnc)
	if err != nil {
		return nil, err
	}
	if len(keyDec) != 32 {
		return nil, fmt.Errorf("expected key length of 32; got %d", len(keyDec))
	}

	var key [32]byte
	copy(key[:], keyDec)

	return &key, nil
}

func generateNonce() (*[nonceSize]byte, error) {
	nonce := new([nonceSize]byte)
	_, err := io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return nil, err
	}

	return nonce, nil
}
