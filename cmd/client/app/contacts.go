package app

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
)

const configFilename = "config.json"

type userDocument struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type document struct {
	User     userDocument   `json:"user"`
	Contacts []userDocument `json:"contacts"`
}

type User struct {
	ID       string
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
	contacts []User
	mu       sync.RWMutex
}

func NewContacts(confDir string) (*Contacts, error) {

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

		id := fmt.Sprintf("%d", rand.IntN(99999))

		doc := document{
			User: userDocument{
				ID:   id,
				Name: "Anonymous",
			},
			Contacts: make([]userDocument, 0),
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

	contacts := make([]User, len(doc.Contacts))
	for i, c := range doc.Contacts {
		contacts[i] = User{
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

func (c *Contacts) LookupContact(id string) (User, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, usr := range c.contacts {
		if usr.ID == id {
			return usr, nil
		}
	}
	return User{}, fmt.Errorf("user not found in contacts")
}

func (c *Contacts) Contacts() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.contacts
}

func (c *Contacts) AddMessage(id string, msg string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, usr := range c.contacts {
		if usr.ID == id {
			usr.Messages = append(usr.Messages, msg)
			return nil
		}
	}

	return fmt.Errorf("user with id %s, not found", id)
}

func (c *Contacts) AddContact(id string, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fullPath := filepath.Join(c.dir, configFilename)
	// Read the entire file first
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", fullPath, err)
	}

	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("decode into doc: %w", err)
	}

	doc.Contacts = append(doc.Contacts, userDocument{ID: id, Name: name})
	c.contacts = append(c.contacts, User{ID: id, Name: name})

	newData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode updates: %w", err)
	}

	if err := os.WriteFile(fullPath, newData, 0644); err != nil {
		return fmt.Errorf("write file %s: %w", fullPath, err)
	}

	return nil
}
