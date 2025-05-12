package main

import (
	"fmt"
	"os"

	"github.com/hamidoujand/echo/cmd/client/chat"
	"github.com/rivo/tview"
)

func main() {
	btnHandler := func(v *tview.TextView) {
		fmt.Fprintln(v, "Button Hit")
	}

	app := chat.NewApp(btnHandler)

	app.WriteMessage("This is a Test!")

	if err := app.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
