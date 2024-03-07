// cryptopasta - basic cryptography examples
//
// Written in 2015 by George Tankersley <george.tankersley@gmail.com>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

// Provides symmetric authenticated encryption using 256-bit AES-GCM with a random nonce.
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/box"
)

// NewEncryptionKey generates a random 256-bit key for Encrypt() and
// Decrypt(). It panics if the source of randomness fails.
func NewEncryptionKey() *[32]byte {
	key := [32]byte{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}
	return &key
}

// DecodeHexKey validates and decodes an encryption key.
func DecodeHexKey(hexKey string) (*[32]byte, error) {
	keyDec, err := hex.DecodeString(hexKey)
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

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Encrypt(plaintext []byte, key *[32]byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Decrypt(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}

// GenerateNonce creates a new random nonce.
func GenerateNonce() (*[24]byte, error) {
	nonce := new([24]byte)
	_, err := io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return nil, err
	}

	return nonce, nil
}

// Maximum size of each encrypted chunk of data. NaCl is recommended for
// encrypting "small" messages, so large data is split into 16KB chunks.
const chunkSize = 16 * 1024 // 16KB

func EncryptNaCl(in io.Reader, publicKey, privateKey *[32]byte) (io.Reader, error) {
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

		nonce, err := GenerateNonce()
		if err != nil {
			return nil, fmt.Errorf("failed generating nonce: %w", err)
		}

		encrypted := box.Seal(nonce[:], buf[:n], nonce, publicKey, privateKey)
		payloadSize := len(encrypted)

		// Write payload size to buffer
		err = binary.Write(out, binary.LittleEndian, uint32(payloadSize))
		if err != nil {
			return nil, fmt.Errorf("failed writing payload size to buffer: %w", err)
		}

		_, err = out.Write(encrypted)
		if err != nil {
			return nil, fmt.Errorf("failed writing encrypted data: %w", err)
		}
	}

	return out, nil
}

func DecryptNaCl(in io.Reader, publicKey, privateKey *[32]byte) (io.Reader, error) {
	out := &bytes.Buffer{}

	for {
		// Read the payload size out of the buffer
		var payloadSize uint32
		err := binary.Read(in, binary.LittleEndian, &payloadSize)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed reading payload size from buffer: %w", err)
		}
		if payloadSize == 0 {
			break
		}

		// Read the payload
		data := make([]byte, payloadSize)
		_, err = io.ReadFull(in, data)
		if err != nil {
			return nil, fmt.Errorf("failed reading payload from buffer: %w", err)
		}

		nonce := data[:24]
		encrypted := data[24:]

		var nonceBuf [24]byte
		copy(nonceBuf[:], nonce)
		decrypted, ok := box.Open(nil, encrypted, &nonceBuf, publicKey, privateKey)
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
