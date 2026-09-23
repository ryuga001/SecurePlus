package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
)

const (
	KeyLength   = 32
	NonceLength = 12

	hkdfInfo     = "dpdp/datadiscovery/credential/v1"
	maxPlaintext = 32 * 1024
)

var (
	ErrKeyInvalid        = errors.New("data discovery master key must be 32 base64-encoded bytes")
	ErrVersionInvalid    = errors.New("data discovery key version must be 1 or greater")
	ErrPlaintextTooLarge = errors.New("credential payload is too large to seal")
	ErrCiphertextInvalid = errors.New("sealed credential is malformed or was sealed with a different key")
)

type SecretBox struct {
	aead    cipher.AEAD
	version int
}

func NewSecretBox(masterKeyBase64 string, version int) (*SecretBox, error) {
	if version < 1 {
		return nil, ErrVersionInvalid
	}

	master, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil || len(master) != KeyLength {
		return nil, ErrKeyInvalid
	}

	derived, err := hkdf.Key(sha256.New, master, nil, hkdfInfo, KeyLength)
	if err != nil {
		return nil, ErrKeyInvalid
	}

	block, err := aes.NewCipher(derived)
	if err != nil {
		return nil, ErrKeyInvalid
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrKeyInvalid
	}

	return &SecretBox{aead: aead, version: version}, nil
}

func (b *SecretBox) Version() int {
	return b.version
}

func (b *SecretBox) Seal(plaintext, aad []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, ErrPlaintextTooLarge
	}
	if len(plaintext) > maxPlaintext {
		return nil, ErrPlaintextTooLarge
	}

	nonce := make([]byte, NonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return b.aead.Seal(nonce, nonce, plaintext, aad), nil
}

func (b *SecretBox) Open(envelope, aad []byte) ([]byte, error) {
	if len(envelope) < NonceLength+b.aead.Overhead() {
		return nil, ErrCiphertextInvalid
	}

	nonce := envelope[:NonceLength]
	ciphertext := envelope[NonceLength:]

	plaintext, err := b.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrCiphertextInvalid
	}

	return plaintext, nil
}

func CredentialAAD(keyVersion, customerID, configurationID int) []byte {
	return []byte("dpdp:dd:cred:v" + strconv.Itoa(keyVersion) +
		":" + strconv.Itoa(customerID) +
		":" + strconv.Itoa(configurationID))
}
