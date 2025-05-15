package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hamidoujand/echo/cmd/client/chat"
)

const url = "ws://localhost:8000/v1/connect"

func main() {
	users := []uuid.UUID{
		uuid.MustParse("8ce5af7a-788c-4c83-8e70-4500b775b359"),
		uuid.MustParse("8a45ec7a-273c-430a-9d90-ac30f94000cd"),
	}

	var ID uuid.UUID

	switch os.Args[1] {
	case "0":
		ID = users[0]
	case "1":
		ID = users[1]
	}

	client := chat.NewClient(ID, url)
	defer client.Close()

	app := chat.NewApp(client)
	name := app.FindName(ID.String())
	writeText := func(name, msg string) {
		app.WriteMessage(name, msg)
	}

	if err := client.Handshake(name, writeText); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	app.WriteMessage("system", "Successfully connected!")

	if err := app.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
