package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

const minToken = 12

// fileFormat holds the values a client derives the same file name and key from.
// They come from settings, and files written with other values can't be read.
type fileFormat struct {
	Magic    string
	NameInfo string
	KeyInfo  string
}

type backupContent struct {
	CreatedAt int64             `json:"created-at"`
	Headers   map[string]string `json:"headers"`
	Config    string            `json:"config"`
}

func (f fileFormat) fileName(token string) (string, error) {
	name, err := hkdf.Key(sha256.New, []byte(token), nil, f.NameInfo, 16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(name), nil
}

func (f fileFormat) cipher(token string) (cipher.AEAD, error) {
	key, err := hkdf.Key(sha256.New, []byte(token), nil, f.KeyInfo, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (f fileFormat) seal(token string, content backupContent) ([]byte, error) {
	if len(token) < minToken {
		return nil, errors.New("token too short")
	}
	aead, err := f.cipher(token)
	if err != nil {
		return nil, err
	}
	plaintext, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := append([]byte(f.Magic), nonce...)
	return aead.Seal(out, nonce, plaintext, []byte(f.Magic)), nil
}

func (f fileFormat) open(token string, data []byte) (backupContent, error) {
	var content backupContent
	aead, err := f.cipher(token)
	if err != nil {
		return content, err
	}
	header := len(f.Magic) + aead.NonceSize()
	if len(data) < header || !bytes.HasPrefix(data, []byte(f.Magic)) {
		return content, errors.New("not a backup file")
	}
	plaintext, err := aead.Open(nil, data[len(f.Magic):header], data[header:], []byte(f.Magic))
	if err != nil {
		return content, err
	}
	err = json.Unmarshal(plaintext, &content)
	return content, err
}
