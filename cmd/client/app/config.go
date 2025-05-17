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
	ID   string
	Name string
}

type Users struct {
	User     User
	Contacts []User
}

type Config struct {
	user     User
	contacts []User
	mu       sync.RWMutex
}

func NewConfig(confDir string) (*Config, error) {

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
		}

		if err := json.NewEncoder(f).Encode(doc); err != nil {
			return nil, fmt.Errorf("encoding cfg to file: %w", err)
		}

		cfg := Config{
			user: User{
				ID:   doc.User.ID,
				Name: doc.User.Name,
			},
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
		contacts[i] = User(c)
	}

	c := Config{
		user: User{
			ID:   doc.User.ID,
			Name: doc.User.Name,
		},
		contacts: contacts,
	}
	return &c, nil
}

func (c *Config) User() User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.user
}

func (c *Config) LookupContact(id string) (User, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, usr := range c.contacts {
		if usr.ID == id {
			return usr, nil
		}
	}
	return User{}, fmt.Errorf("user not found in contacts")
}

func (c *Config) Contacts() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.contacts
}

func (c *Config) AddContact(user User) error {
	return nil
}
