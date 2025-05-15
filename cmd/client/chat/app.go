package chat

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
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
}

func NewApp(client *Client) *App {

	app := tview.NewApplication()

	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")
	list.AddItem("John Doe", "8ce5af7a-788c-4c83-8e70-4500b775b359", '1', nil)
	list.AddItem("Jane Doe", "8a45ec7a-273c-430a-9d90-ac30f94000cd", '2', nil)

	// -------------------------------------------------------------------------

	textView := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textView.SetBorder(true)
	textView.SetTitle("chat")

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
	}

	button.SetSelectedFunc(a.buttonHandler)

	return a
}

func (a *App) Run() error {
	return a.app.SetRoot(a.flex, true).EnableMouse(true).Run()
}

func (a *App) FindName(id string) string {
	for i := range a.list.GetItemCount() {
		name, toId := a.list.GetItemText(i)
		if toId == id {
			return name
		}
	}
	return ""
}

func (a *App) WriteMessage(name string, msg string) {
	a.textView.ScrollToEnd()
	fmt.Fprintln(a.textView, "--------------------------------------")
	fmt.Fprintln(a.textView, name+": "+msg)
}

func (a *App) buttonHandler() {
	_, receiverID := a.list.GetItemText(a.list.GetCurrentItem())

	uid, err := uuid.Parse(receiverID)
	if err != nil {
		a.WriteMessage("system", fmt.Sprintf("error parsing TO id: %s", err))
		return
	}

	msg := a.textArea.GetText()
	if msg == "" {
		return
	}

	if err := a.client.Send(uid, msg); err != nil {
		a.WriteMessage("system", fmt.Sprintf("sending message failed: %s", err))
		return
	}

	a.textArea.SetText("", false)
	a.WriteMessage("You", msg)
}
