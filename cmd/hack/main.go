package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hamidoujand/echo/signature"
)

var privatePath = filepath.Join("infra", "private.ecdsa")

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if err := generateECDSAPrivate(privatePath); err != nil {
		return err
	}

	private, err := crypto.LoadECDSA(privatePath)
	if err != nil {
		return fmt.Errorf("loadECDSA: %w", err)
	}

	addr := crypto.PubkeyToAddress(private.PublicKey)
	fmt.Printf("ID: %s\n", addr.Hex())

	//sign data
	data := struct {
		FromID string `json:"fromID"`
		ToID   string `json:"toID"`
		Text   string `json:"text"`
		Nonce  uint64 `json:"nonce"`
	}{
		FromID: addr.String(),
		ToID:   "27362",
		Text:   "Hello John",
		Nonce:  1,
	}

	v, r, s, err := signature.Sign(data, private)
	if err != nil {
		return fmt.Errorf("signing data: %w", err)
	}

	fmt.Println("V:", v)
	fmt.Println("R:", r)
	fmt.Println("S:", s)

	//==========================================================================
	id, err := signature.FromAddress(data, v, r, s)
	if err != nil {
		return fmt.Errorf("fromAddress: %w", err)
	}

	fmt.Println("ID2:", id)

	return nil
}

func generateECDSAPrivate(path string) error {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("generatePrivateKey: %w", err)
	}

	if err := crypto.SaveECDSA(path, privateKey); err != nil {
		return fmt.Errorf("saveECDSA: %w", err)
	}
	return nil
}
