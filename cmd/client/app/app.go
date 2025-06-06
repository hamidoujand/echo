package app

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	app      *tview.Application
	flex     *tview.Flex
	textView *tview.TextView
	button   *tview.Button
	list     *tview.List
	textArea *tview.TextArea
	client   *Client
	db       *Database
}

func New(client *Client, db *Database) *App {

	app := tview.NewApplication()

	// -------------------------------------------------------------------------
	textView := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf("*** %s ***", db.MyAccount().ID))
	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")
	list.SetChangedFunc(func(index int, name, id string, shortcut rune) {
		textView.Clear()

		commonID := common.HexToAddress(id)

		usr, err := db.LookupContact(commonID)
		if err != nil {
			textView.ScrollToEnd()
			fmt.Fprintln(textView, "--------------------------------------")
			fmt.Fprintln(textView, "system: "+err.Error())
			return
		}
		for i, msg := range usr.Messages {
			fmt.Fprintln(textView, string(msg.Text))
			if i < len(usr.Messages)-1 {
				fmt.Fprintln(textView, "--------------------------------------")
			}
		}

		list.SetItemText(index, usr.Name, usr.ID.String())
	})

	users := db.Contacts()
	for i, c := range users {
		shortcut := rune(i + 49)
		list.AddItem(c.Name, c.ID.Hex(), shortcut, nil)
	}

	// -------------------------------------------------------------------------

	button := tview.NewButton("SUBMIT")
	button.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetBorder(true)
	button.SetBorderPadding(0, 1, 0, 0)
	button.SetBorderColor(tcell.ColorGreen)

	// -------------------------------------------------------------------------

	textArea := tview.NewTextArea()
	textArea.SetWrap(false)
	textArea.SetPlaceholder("Enter message here...")
	textArea.SetBorder(true)
	textArea.SetBorderPadding(0, 0, 1, 0)

	// -------------------------------------------------------------------------

	flex := tview.NewFlex().
		AddItem(list, 20, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(textView, 0, 5, false).
			AddItem(tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(textArea, 0, 90, false).
				AddItem(button, 0, 10, false),
				0, 1, false),
			0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlQ:
			app.Stop()
			return nil
		}
		return event
	})

	a := &App{
		app:      app,
		flex:     flex,
		textView: textView,
		button:   button,
		textArea: textArea,
		client:   client,
		list:     list,
		db:       db,
	}

	button.SetSelectedFunc(a.buttonHandler)

	return a
}

func (a *App) Run() error {
	return a.app.SetRoot(a.flex, true).EnableMouse(true).Run()
}

func (a *App) WriteMessage(id string, msg message) {
	a.textView.ScrollToEnd()

	switch id {
	case "system":
		fmt.Fprintln(a.textView, "--------------------------------------")
		fmt.Fprintf(a.textView, "%s: %s\n", msg.Name, string(msg.Text))
	default:
		idx := a.list.GetCurrentItem()
		_, currentID := a.list.GetItemText(idx)
		if currentID == "" {
			fmt.Fprintln(a.textView, "--------------------------------------")
			fmt.Fprintln(a.textView, "id not found: "+id)
			return
		}

		if id == currentID {
			fmt.Fprintln(a.textView, "--------------------------------------")
			fmt.Fprintf(a.textView, "%s: %s\n", msg.Name, string(msg.Text))
			return
		}

		for i := range a.list.GetItemCount() {
			name, idStr := a.list.GetItemText(i)
			if idStr == id {
				a.list.SetItemText(i, "* "+name, idStr)
				a.app.Draw()
				return
			}
		}

	}

}

func (a *App) buttonHandler() {
	if len(a.db.contacts) == 0 {
		return
	}
	_, receiverID := a.list.GetItemText(a.list.GetCurrentItem())

	msg := a.textArea.GetText()
	if msg == "" {
		return
	}

	id := common.HexToAddress(receiverID)

	if err := a.client.Send(id, []byte(msg)); err != nil {
		msg := message{
			Name: "system",
			Text: fmt.Appendf(nil, "sending message failed: %s", err),
		}
		a.WriteMessage("system", msg)
		return
	}

	a.textArea.SetText("", false)
}

func (a *App) UpdateContact(id string, name string) {
	shortcut := rune(a.list.GetItemCount() + 49)
	a.list.AddItem(name, id, shortcut, nil)
}
