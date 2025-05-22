package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const idFilename = "private.ecdsa"

func NewID(confDir string) (common.Address, error) {
	if err := os.MkdirAll(filepath.Join(confDir, "id"), 0755); err != nil {
		return common.Address{}, fmt.Errorf("mkdirAll: %w", err)
	}

	filename := filepath.Join(confDir, "id", idFilename)
	var id common.Address
	_, err := os.Stat(filename)
	switch {
	case err != nil:
		var err error
		id, err = createKeyID(filename)
		if err != nil {
			return common.Address{}, fmt.Errorf("createKeyID: %w", err)
		}
	default:
		var err error
		id, err = readKeyID(filename)
		if err != nil {
			return common.Address{}, fmt.Errorf("readKeyID: %w", err)
		}
	}

	return id, nil
}

func createKeyID(filename string) (common.Address, error) {
	private, err := crypto.GenerateKey()
	if err != nil {
		return common.Address{}, fmt.Errorf("generateKey: %w", err)
	}

	if err := crypto.SaveECDSA(filename, private); err != nil {
		return common.Address{}, fmt.Errorf("saveECDSA: %w", err)
	}

	addr := crypto.PubkeyToAddress(private.PublicKey)
	return addr, nil
}

func readKeyID(filename string) (common.Address, error) {
	private, err := crypto.LoadECDSA(filename)
	if err != nil {
		return common.Address{}, fmt.Errorf("loadECDSA: %w", err)
	}
	addr := crypto.PubkeyToAddress(private.PublicKey)
	return addr, nil
}
