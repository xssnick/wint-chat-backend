package cryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

type AESGCM struct {
	gcm cipher.AEAD
}

func (a *AESGCM) Encrypt(text []byte) ([]byte, error) {
	if len(text) == 0 {
		return nil, errors.New("input data is empty")
	}

	nonce := make([]byte, a.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return a.gcm.Seal(nonce, nonce, text, nil), nil
}

func (a *AESGCM) Decrypt(text []byte) ([]byte, error) {
	if len(text) == 0 {
		return nil, errors.New("input data is empty")
	}

	nonceSize := a.gcm.NonceSize()
	if len(text) <= nonceSize {
		return nil, errors.New("input data len is smaller than size of nonce")
	}

	return a.gcm.Open(nil, text[:nonceSize], text[nonceSize:], nil)
}

func NewAESGCM(key []byte) (*AESGCM, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	return &AESGCM{gcm}, nil
}
