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
	id, err := app.NewID(configDir)
	if err != nil {
		return fmt.Errorf("newID: %w", err)
	}

	db, err := app.NewDatabase(configDir, id.Address)
	if err != nil {
		return fmt.Errorf("newDatabase: %w", err)
	}

	client := app.NewClient(id, url, db)
	defer client.Close()

	a := app.New(client, db)

	if err := client.Handshake(db.MyAccount().Name, a.WriteMessage, a.UpdateContact); err != nil {
		return fmt.Errorf("client handshake failed: %w", err)
	}

	if err := a.Run(); err != nil {
		return fmt.Errorf("application run failed: %w", err)
	}
	return nil
}
