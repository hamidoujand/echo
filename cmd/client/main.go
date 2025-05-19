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
	contacts, err := app.NewContacts(configDir)
	if err != nil {
		return fmt.Errorf("newContacts: %w", err)
	}

	usr := contacts.My()
	client := app.NewClient(usr.ID, url, contacts)
	defer client.Close()

	a := app.New(client, contacts)

	// uiWriter := func(id, msg string) {
	// 	a.WriteMessage(id, msg)
	// }

	// updateContacts := func(id, name string) {
	// 	a.UpdateContact(id, name)
	// }

	if err := client.Handshake(usr.Name, a.WriteMessage, a.UpdateContact); err != nil {
		return fmt.Errorf("client handshake failed: %w", err)
	}

	a.WriteMessage("system", "Successfully connected!")

	if err := a.Run(); err != nil {
		return fmt.Errorf("application run failed: %w", err)
	}
	return nil
}
