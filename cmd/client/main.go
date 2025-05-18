package main

import (
	"fmt"
	"os"

	"github.com/hamidoujand/echo/cmd/client/app"
)

const (
	url       = "ws://localhost:8000/v1/connect"
	configDir = "infra"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	conf, err := app.NewConfig(configDir)
	if err != nil {
		return fmt.Errorf("newConfig: %w", err)
	}

	usr := conf.User()
	client := app.NewClient(usr.ID, url, conf)
	defer client.Close()

	a := app.New(client, conf)

	uiWriter := func(name, msg string) {
		a.WriteMessage(name, msg)
	}

	updateContacts := func(id, name string) {
		a.UpdateContact(id, name)
	}

	if err := client.Handshake(usr.Name, uiWriter, updateContacts); err != nil {
		return fmt.Errorf("client handshake failed: %w", err)
	}

	a.WriteMessage("system", "Successfully connected!")

	if err := a.Run(); err != nil {
		return fmt.Errorf("application run failed: %w", err)
	}
	return nil
}
