package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	const url = "ws://localhost:8000/v1/connect"
	socket, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer socket.Close()

	_, msg, err := socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("readMessage: %w", err)
	}

	if string(msg) != "Hello" {
		return fmt.Errorf("expected msg to be Hello, got %s", string(msg))
	}

	//send
	usr := map[string]any{
		"ID":   uuid.New(),
		"Name": "John Doe",
	}

	bs, err := json.Marshal(usr)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := socket.WriteMessage(websocket.TextMessage, bs); err != nil {
		return fmt.Errorf("writeMessage: %w", err)
	}

	_, msg, err = socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("readMessage: %w", err)
	}

	fmt.Println(string(msg))

	return nil
}
