package chat

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	app      *tview.Application
	flex     *tview.Flex
	textView *tview.TextView
}

type ButtonHandler func(textView *tview.TextView)

func NewApp(bthHandler ButtonHandler) *App {

	app := tview.NewApplication()

	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")
	list.AddItem("John Doe", "", '1', nil)
	list.AddItem("Jane Doe", "", '2', nil)

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
	button.SetSelectedFunc(func() {
		bthHandler(textView)
	})
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

	return &App{
		app:      app,
		flex:     flex,
		textView: textView,
	}
}

func (v *App) Run() error {
	return v.app.SetRoot(v.flex, true).EnableMouse(true).Run()
}

func (v *App) WriteMessage(msg string) {
	v.textView.ScrollToEnd()
	fmt.Fprintln(v.textView, msg)
}

func (v *App) ClearMessage() {
	v.textView.Clear()
}
