package app

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const idFilename = "private.ecdsa"
const encryptionFilename = "private.rsa"

type ID struct {
	Address      common.Address
	ECDSAKey     *ecdsa.PrivateKey
	RSAKey       *rsa.PrivateKey
	RSAPublicKey string
}

func NewID(confDir string) (ID, error) {
	if err := os.MkdirAll(filepath.Join(confDir, "id"), 0755); err != nil {
		return ID{}, fmt.Errorf("mkdirAll: %w", err)
	}

	filename := filepath.Join(confDir, "id", idFilename)
	var address common.Address
	var privateECDSA *ecdsa.PrivateKey
	_, err := os.Stat(filename)
	switch {
	case err != nil:
		var err error
		address, privateECDSA, err = createKeyID(filename)
		if err != nil {
			return ID{}, fmt.Errorf("createKeyID: %w", err)
		}
	default:
		var err error
		address, privateECDSA, err = readKeyID(filename)
		if err != nil {
			return ID{}, fmt.Errorf("readKeyID: %w", err)
		}
	}

	encryptKeyFile := filepath.Join(confDir, "id", encryptionFilename)
	var privateRSA *rsa.PrivateKey
	_, err = os.Stat(encryptKeyFile)
	switch {
	case err != nil:
		var err error
		privateRSA, err = createEncryptKey(encryptKeyFile)
		if err != nil {
			return ID{}, fmt.Errorf("createEncryptKey: %w", err)
		}
	default:
		var err error
		privateRSA, err = readEncryptKey(encryptKeyFile)
		if err != nil {
			return ID{}, fmt.Errorf("readEncryptKey: %w", err)
		}
	}

	//public key
	bs, err := x509.MarshalPKIXPublicKey(&privateRSA.PublicKey)
	if err != nil {
		return ID{}, fmt.Errorf("marshalling Public Key: %w", err)
	}

	publicPEM := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: bs,
	}

	var builder strings.Builder
	if err := pem.Encode(&builder, &publicPEM); err != nil {
		return ID{}, fmt.Errorf("encode public key: %w", err)
	}

	return ID{Address: address, ECDSAKey: privateECDSA, RSAKey: privateRSA, RSAPublicKey: builder.String()}, nil
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

func createEncryptKey(filename string) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("creating private key file: %w", err)
	}

	defer file.Close()

	block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	if err := pem.Encode(file, &block); err != nil {
		return nil, fmt.Errorf("encoding to pem: %w", err)
	}

	return privateKey, nil
}

func readEncryptKey(filename string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid PEM block type: %s", block.Type)
	}

	private, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	return private, nil
}
