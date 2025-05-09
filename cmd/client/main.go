package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	//==========================================================================
	go func() {
		for {
			_, msg, err = socket.ReadMessage()
			if err != nil {
				fmt.Printf("readMessage: %s\n", err)
				return
			}

			//create the inMessage
			var out outMessage
			if err := json.Unmarshal(msg, &out); err != nil {
				fmt.Printf("failed to unmarshal: %s\n", err)
				return
			}

			fmt.Printf("%s:%s\n", out.From.Name, out.Text)
		}
	}()
	fmt.Print("user:Message > ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	data := strings.Split(input, ":")
	if len(data) != 2 {
		return fmt.Errorf("length of input must be 2, got %d", len(data))
	}

	idString, m := data[0], data[1]
	id, err := strconv.Atoi(idString)
	if err != nil {
		return fmt.Errorf("atoi: %w", err)
	}
	message := strings.TrimSpace(m)

	users := []uuid.UUID{
		uuid.MustParse("890c122e-5c97-45b1-b5fd-2cb1ae38ba4a"),
		uuid.MustParse("608e677b-e408-4a12-a061-adccf40c628a"),
	}

	from := users[0]
	to := users[1]
	if id == 1 {
		from = users[1]
		to = users[0]
	}

	in := inMessage{
		FromID: from,
		ToID:   to,
		Text:   message,
	}

	bs, err = json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshaling message to send: %w", err)
	}

	if err := socket.WriteMessage(websocket.TextMessage, bs); err != nil {
		return fmt.Errorf("write message into socket: %w", err)
	}

	return nil
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type inMessage struct {
	FromID uuid.UUID `json:"fromID"`
	ToID   uuid.UUID `json:"toID"`
	Text   string    `json:"text"`
}

type outMessage struct {
	From user   `json:"from"`
	To   user   `json:"to"`
	Text string `json:"text"`
}
