package rsa

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

var (
	ErrInvalidPrivateKey = fmt.Errorf("invalid private key")
	ErrInvalidPublicKey  = fmt.Errorf("invalid public key")
)

type PrivateKey struct {
	*rsa.PrivateKey
	PrivateKeyPEM string
	PublicKeyPEM  string
}

func GeneratePrivateKey(bits int) (*PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return nil, err
	}
	privateKeyPEM := buf.String()
	buf.Reset()
	if err := pem.Encode(&buf, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	}); err != nil {
		return nil, err
	}
	publicKeyPEM := buf.String()
	return &PrivateKey{privateKey, privateKeyPEM, publicKeyPEM}, nil
}

func ParsePrivateKey(privateKeyPEM string) (*PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, ErrInvalidPrivateKey
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err := pem.Encode(buf, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	}); err != nil {
		return nil, err
	}
	publicKeyPEM := buf.String()
	return &PrivateKey{privateKey, privateKeyPEM, publicKeyPEM}, nil
}

func (k *PrivateKey) Sign(msg []byte) ([]byte, error) {
	hasher := sha256.New()
	hasher.Write(msg)
	digest := hasher.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, k.PrivateKey, crypto.SHA256, digest)
}

func (k *PrivateKey) SignBase64(msg []byte) (string, error) {
	sign, err := k.Sign(msg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}

func (k *PrivateKey) Decrypt(ciphertext []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, k.PrivateKey, ciphertext)
}

func (k *PrivateKey) DecryptBase64(s string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return k.Decrypt(ciphertext)
}

type PublicKey struct {
	*rsa.PublicKey
}

func ParsePublicKey(publicKeyPEM string) (*PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, ErrInvalidPublicKey
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &PublicKey{publicKey}, nil
}

func (k *PublicKey) Encrypt(msg []byte) ([]byte, error) {
	var block, out []byte
	var err error
	size := k.Size() - 11
	for len(msg) > 0 {
		if len(msg) > size {
			block = msg[:size]
			msg = msg[size:]
		} else {
			block = msg
			msg = nil
		}
		block, err = rsa.EncryptPKCS1v15(rand.Reader, k.PublicKey, block)
		if err != nil {
			return nil, err
		}
		out = append(out, block...)
	}
	return out, nil
}

func (k *PublicKey) EncryptBase64(msg []byte) (string, error) {
	ciphertext, err := k.Encrypt(msg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
