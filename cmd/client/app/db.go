package app

import (
	"bufio"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const dbFilename = "data.json"
const chatHistoryDir = "messages"

type profile struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}

type contact struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
	// Nonce for messages YOU send to THIS contact.
	OutgoingNonce uint64 `json:"outgoingNonce"`
	// Nonce for messages THIS contact sends to YOU
	IncomingNonce uint64 `json:"incomingNonce"`
	Key           []byte `json:"key"`
}

type account struct {
	MyAccount profile   `json:"myAccount"`
	Contacts  []contact `json:"contacts"`
}

type User struct {
	ID            common.Address
	Name          string
	OutgoingNonce uint64
	IncomingNonce uint64
	Key           []byte
	Messages      [][]byte
}

type Users struct {
	User     User
	Contacts []User
}

type Database struct {
	myAccount User
	dir       string
	contacts  map[common.Address]User
	mu        sync.RWMutex
}

func NewDatabase(confDir string, myAccountID common.Address) (*Database, error) {
	chatHistoryPath := filepath.Join(confDir, chatHistoryDir)
	if err := os.MkdirAll(chatHistoryPath, 0755); err != nil {
		return nil, fmt.Errorf("chatHistory mkdirAll: %w", err)
	}

	fullPath := filepath.Join(confDir, dbFilename)

	//file not exists
	if _, err := os.Stat(fullPath); err != nil {
		if err := os.MkdirAll(confDir, 0755); err != nil {
			return nil, fmt.Errorf("mkdirAll: %w", err)
		}

		f, err := os.Create(fullPath)
		if err != nil {
			return nil, fmt.Errorf("create file %s: %w", fullPath, err)
		}
		defer f.Close()

		doc := account{
			MyAccount: profile{
				ID:   myAccountID,
				Name: "Anonymous",
			},
			Contacts: []contact{
				{
					ID:   common.Address{},
					Name: "Sample Contact",
				},
			},
		}

		if err := json.NewEncoder(f).Encode(doc); err != nil {
			return nil, fmt.Errorf("encoding cfg to file: %w", err)
		}

		db := Database{
			myAccount: User{
				ID:   doc.MyAccount.ID,
				Name: doc.MyAccount.Name,
			},
			dir: confDir,
		}

		return &db, nil
	}
	//file exists
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", fullPath, err)
	}
	defer f.Close()

	var acc account
	if err := json.NewDecoder(f).Decode(&acc); err != nil {
		return nil, fmt.Errorf("decode into account: %w", err)
	}

	if acc.MyAccount.ID != myAccountID {
		return nil, errors.New("id mismatch")
	}

	contacts := make(map[common.Address]User, len(acc.Contacts))
	for _, c := range acc.Contacts {
		// var pk *rsa.PublicKey
		// if c.Key != "" {
		// 	block, _ := pem.Decode([]byte(c.Key))
		// 	if block == nil {
		// 		return nil, fmt.Errorf("decoding contact's public key into pem block: %w", err)
		// 	}

		// 	var err error
		// 	pk, err = x509.ParsePKCS1PublicKey(block.Bytes)
		// 	if err != nil {
		// 		return nil, fmt.Errorf("parsing public key: %w", err)
		// 	}

		// }
		contacts[c.ID] = User{
			ID:            c.ID,
			Name:          c.Name,
			OutgoingNonce: c.OutgoingNonce,
			IncomingNonce: c.IncomingNonce,
			Key:           c.Key,
		}
	}

	c := Database{
		myAccount: User{
			ID:   acc.MyAccount.ID,
			Name: acc.MyAccount.Name,
		},
		contacts: contacts,
	}
	return &c, nil
}

func (db *Database) MyAccount() User {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.myAccount
}

func (db *Database) LookupContact(id common.Address) (User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	usr, ok := db.contacts[id]
	if !ok {
		return User{}, fmt.Errorf("contact with id %s not found", id.String())
	}

	if len(usr.Messages) == 0 {
		//read messages from disk

	}

	return usr, nil
}

func (db *Database) Contacts() []User {
	db.mu.RLock()
	defer db.mu.RUnlock()
	users := make([]User, 0, len(db.contacts))

	for _, usr := range db.contacts {
		users = append(users, usr)
	}

	return users
}

func (db *Database) AddMessage(id common.Address, msg []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	usr, ok := db.contacts[id]
	if !ok {
		return fmt.Errorf("contact with id %s not found", id.String())
	}

	usr.Messages = append(usr.Messages, msg)
	db.contacts[id] = usr

	if err := db.writeMessage(id, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func (db *Database) AddContact(id common.Address, name string) (User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	fullPath := filepath.Join(db.dir, dbFilename)

	// Read the entire db file first
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return User{}, fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var acc account
	if err := json.Unmarshal(data, &acc); err != nil {
		return User{}, fmt.Errorf("decode into account: %w", err)
	}

	acc.Contacts = append(acc.Contacts, contact{ID: id, Name: name})
	db.contacts[id] = User{
		ID:   id,
		Name: name,
	}

	newData, err := json.Marshal(acc)
	if err != nil {
		return User{}, fmt.Errorf("encode updates: %w", err)
	}

	if err := os.WriteFile(fullPath, newData, 0644); err != nil {
		return User{}, fmt.Errorf("write file %s: %w", fullPath, err)
	}

	u := User{
		ID:   id,
		Name: name,
	}
	return u, nil
}

func (db *Database) ReadMessage(id common.Address) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	historyFile := filepath.Join(db.dir, chatHistoryDir, id.Hex()+".msg")

	usr, ok := db.contacts[id]
	if !ok {
		return fmt.Errorf("contact with id %s not found", id.String())
	}

	if len(usr.Messages) > 0 {
		return nil
	}
	f, err := os.Open(historyFile)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Bytes()
		usr.Messages = append(usr.Messages, txt)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("while scanning file: %w", err)
	}

	db.contacts[id] = usr

	return nil
}

func (db *Database) writeMessage(id common.Address, msg []byte) error {
	filename := filepath.Join(db.dir, chatHistoryDir, id.String()+".msg")
	_, err := os.Stat(filename)
	var f *os.File

	if err != nil {
		var err error
		f, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("create file %s: %w", filename, err)
		}
	} else {
		var err error
		f, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("openFile %s: %w", filename, err)
		}
	}
	defer f.Close()

	if _, err := f.Write(fmt.Appendf(nil, "%s\n", msg)); err != nil {
		return fmt.Errorf("writeString: %w", err)
	}
	return nil
}

func (db *Database) UpdateOutgoingNonce(id common.Address, appNonce uint64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	//update the in-memory cache
	u, ok := db.contacts[id]
	if !ok {
		return fmt.Errorf("user with id %s, not found", id.Hex())
	}

	u.OutgoingNonce = appNonce

	db.contacts[id] = u

	//update the disk

	fullPath := filepath.Join(db.dir, dbFilename)

	// Read the entire db file first
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var acc account
	if err := json.Unmarshal(data, &acc); err != nil {
		return fmt.Errorf("decode into account: %w", err)
	}

	//TODO: change to map[string]Account for performance.
	for i := range acc.Contacts {
		if acc.Contacts[i].ID == id {
			acc.Contacts[i].OutgoingNonce = appNonce
			break
		}
	}

	newData, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("encode updates: %w", err)
	}

	if err := os.WriteFile(fullPath, newData, 0644); err != nil {
		return fmt.Errorf("write file %s: %w", fullPath, err)
	}

	return nil
}

func (db *Database) UpdateIncomingNonce(id common.Address, contactNonce uint64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	//update the in-memory cache
	u, ok := db.contacts[id]
	if !ok {
		return fmt.Errorf("user with id %s, not found", id.Hex())
	}

	u.IncomingNonce = contactNonce

	db.contacts[id] = u

	//update the disk

	fullPath := filepath.Join(db.dir, dbFilename)

	// Read the entire db file first
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var acc account
	if err := json.Unmarshal(data, &acc); err != nil {
		return fmt.Errorf("decode into account: %w", err)
	}

	//TODO: change to map[string]Account for performance.
	for i := range acc.Contacts {
		if acc.Contacts[i].ID == id {
			acc.Contacts[i].IncomingNonce = contactNonce
			break
		}
	}

	newData, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("encode updates: %w", err)
	}

	if err := os.WriteFile(fullPath, newData, 0644); err != nil {
		return fmt.Errorf("write file %s: %w", fullPath, err)
	}

	return nil
}

func (db *Database) UpdateContactKey(id common.Address, key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	//update the in-memory cache
	u, ok := db.contacts[id]
	if !ok {
		return fmt.Errorf("user with id %s, not found", id.Hex())
	}

	u.Key = key

	db.contacts[id] = u

	//update the disk

	fullPath := filepath.Join(db.dir, dbFilename)

	// Read the entire db file first
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var acc account
	if err := json.Unmarshal(data, &acc); err != nil {
		return fmt.Errorf("decode into account: %w", err)
	}

	//TODO: change to map[string]Account for performance.
	for i := range acc.Contacts {
		if acc.Contacts[i].ID == id {
			acc.Contacts[i].Key = key
			break
		}
	}

	newData, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("encode updates: %w", err)
	}

	if err := os.WriteFile(fullPath, newData, 0644); err != nil {
		return fmt.Errorf("write file %s: %w", fullPath, err)
	}

	return nil
}

func parseRSAPublicKey(key []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("decoding contact's public key into pem block")
	}

	pk, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	rsaPublicKey, ok := pk.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not a *rsa.PublicKey: %T", pk)
	}

	return rsaPublicKey, nil
}
