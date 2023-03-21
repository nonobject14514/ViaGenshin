package core

import (
	"encoding/base64"
	"fmt"

	"github.com/Jx2f/ViaGenshin/internal/config"
	"github.com/Jx2f/ViaGenshin/pkg/crypto/ec2b"
	"github.com/Jx2f/ViaGenshin/pkg/crypto/rsa"
)

type Keys struct {
	SharedKey  *ec2b.Ec2b
	ServerKey  *rsa.PrivateKey
	ClientKeys map[uint32]*rsa.PrivateKey
}

func NewKeysFromConfig(config *config.ConfigKeys) (*Keys, error) {
	p, err := base64.StdEncoding.DecodeString(config.SharedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid shared key: %w", err)
	}
	sharedKey, err := ec2b.LoadKey(p)
	if err != nil {
		return nil, fmt.Errorf("invalid shared key: %w", err)
	}
	serverKey, err := rsa.ParsePrivateKey(config.ServerKey)
	if err != nil {
		return nil, fmt.Errorf("invalid server key: %w", err)
	}
	clientKeys := make(map[uint32]*rsa.PrivateKey)
	for id, key := range config.ClientKeys {
		clientKeys[id], err = rsa.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("invalid client key for %d: %w", id, err)
		}
	}
	return &Keys{
		SharedKey:  sharedKey,
		ServerKey:  serverKey,
		ClientKeys: clientKeys,
	}, nil
}
