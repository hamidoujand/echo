package app

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const idFilename = "private.ecdsa"

func NewID(confDir string) (common.Address, *ecdsa.PrivateKey, error) {
	if err := os.MkdirAll(filepath.Join(confDir, "id"), 0755); err != nil {
		return common.Address{}, nil, fmt.Errorf("mkdirAll: %w", err)
	}

	filename := filepath.Join(confDir, "id", idFilename)
	var id common.Address
	var private *ecdsa.PrivateKey
	_, err := os.Stat(filename)
	switch {
	case err != nil:
		var err error
		id, private, err = createKeyID(filename)
		if err != nil {
			return common.Address{}, nil, fmt.Errorf("createKeyID: %w", err)
		}
	default:
		var err error
		id, private, err = readKeyID(filename)
		if err != nil {
			return common.Address{}, nil, fmt.Errorf("readKeyID: %w", err)
		}
	}

	return id, private, nil
}

func createKeyID(filename string) (common.Address, *ecdsa.PrivateKey, error) {
	private, err := crypto.GenerateKey()
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("generateKey: %w", err)
	}

	if err := crypto.SaveECDSA(filename, private); err != nil {
		return common.Address{}, nil, fmt.Errorf("saveECDSA: %w", err)
	}

	addr := crypto.PubkeyToAddress(private.PublicKey)
	return addr, private, nil
}

func readKeyID(filename string) (common.Address, *ecdsa.PrivateKey, error) {
	private, err := crypto.LoadECDSA(filename)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("loadECDSA: %w", err)
	}
	addr := crypto.PubkeyToAddress(private.PublicKey)
	return addr, private, nil
}
