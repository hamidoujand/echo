package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const configFilename = "config.json"

type userDocument struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}

type document struct {
	User     userDocument   `json:"user"`
	Contacts []userDocument `json:"contacts"`
}

type User struct {
	ID       common.Address
	Name     string
	Messages []string
}

type Users struct {
	User     User
	Contacts []User
}

type Contacts struct {
	me       User
	dir      string
	contacts map[common.Address]User
	mu       sync.RWMutex
}

func NewContacts(confDir string, id common.Address) (*Contacts, error) {
	chatHistoryPath := filepath.Join(confDir, "contacts")
	if err := os.MkdirAll(chatHistoryPath, 0755); err != nil {
		return nil, fmt.Errorf("chatHistory mkdirAll: %w", err)
	}

	fullPath := filepath.Join(confDir, configFilename)

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

		doc := document{
			User: userDocument{
				ID:   id,
				Name: "Anonymous",
			},
			Contacts: []userDocument{
				{
					ID:   common.Address{},
					Name: "Sample Contact",
				},
			},
		}

		if err := json.NewEncoder(f).Encode(doc); err != nil {
			return nil, fmt.Errorf("encoding cfg to file: %w", err)
		}

		cfg := Contacts{
			me: User{
				ID:   doc.User.ID,
				Name: doc.User.Name,
			},
			dir: confDir,
		}

		return &cfg, nil
	}
	//file exists
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", fullPath, err)
	}
	defer f.Close()

	var doc document
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode into doc: %w", err)
	}

	if doc.User.ID != id {
		return nil, errors.New("id mismatch")
	}

	contacts := make(map[common.Address]User, len(doc.Contacts))
	for _, c := range doc.Contacts {
		contacts[c.ID] = User{
			ID:   c.ID,
			Name: c.Name,
		}
	}

	c := Contacts{
		me: User{
			ID:   doc.User.ID,
			Name: doc.User.Name,
		},
		contacts: contacts,
	}
	return &c, nil
}

func (c *Contacts) My() User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.me
}

func (c *Contacts) LookupContact(id common.Address) (User, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	usr, ok := c.contacts[id]
	if !ok {
		return User{}, fmt.Errorf("contact with id %s not found", id.String())
	}
	return usr, nil
}

func (c *Contacts) Contacts() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	users := make([]User, 0, len(c.contacts))

	for _, usr := range c.contacts {
		users = append(users, usr)
	}

	return users
}

func (c *Contacts) AddMessage(id common.Address, msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	usr, ok := c.contacts[id]
	if !ok {
		return fmt.Errorf("contact with id %s not found", id.String())
	}

	usr.Messages = append(usr.Messages, msg)
	c.contacts[id] = usr

	if err := c.writeMessage(id, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func (c *Contacts) AddContact(id common.Address, name string) (User, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fullPath := filepath.Join(c.dir, configFilename)
	// Read the entire file first
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return User{}, fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return User{}, fmt.Errorf("decode into doc: %w", err)
	}

	doc.Contacts = append(doc.Contacts, userDocument{ID: id, Name: name})
	c.contacts[id] = User{
		ID:   id,
		Name: name,
	}

	newData, err := json.Marshal(doc)
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

func (c *Contacts) ReadMessage(id common.Address) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	historyFile := filepath.Join(c.dir, "contacts", id.Hex()+".msg")

	usr, ok := c.contacts[id]
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
		txt := scanner.Text()
		usr.Messages = append(usr.Messages, txt)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("while scanning file: %w", err)
	}

	c.contacts[id] = usr

	return nil
}

func (c *Contacts) writeMessage(id common.Address, msg string) error {
	filename := filepath.Join(c.dir, "contacts", id.String()+".msg")
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

	if _, err := f.WriteString(msg); err != nil {
		return fmt.Errorf("writeString: %w", err)
	}
	return nil
}
