package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type contactItem struct {
	name    string
	lastMsg string
}

func (c contactItem) Title() string       { return c.name }
func (c contactItem) FilterValue() string { return c.name }
func (c contactItem) Description() string { return c.lastMsg }

var dummyContacts = []list.Item{
	contactItem{name: "Alice", lastMsg: "Hey there!"},
	contactItem{name: "Bob", lastMsg: "Meeting at 3pm"},
	contactItem{name: "Charlie", lastMsg: "Sent the files"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

type model struct {
	contacts   list.Model
	messages   viewport.Model
	input      textinput.Model
	windowSize tea.WindowSizeMsg
	history    string
}

func initialModel() model {
	// Contact list
	contactList := list.New(dummyContacts, list.NewDefaultDelegate(), 0, 0)
	contactList.Title = "Contacts"
	contactList.SetShowHelp(false)

	// Viewport
	msgView := viewport.New(0, 0)
	welcomeMsg := "Welcome to Echo.\nSelect a contact from left and start chatting.\n"
	msgView.SetContent(welcomeMsg)

	// Text input
	input := textinput.New()
	input.Placeholder = "Type your message..."
	input.Focus()
	input.CharLimit = 200
	input.Prompt = "> "
	input.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	return model{
		contacts: contactList,
		messages: msgView,
		input:    input,
		history:  welcomeMsg,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.updateLayout()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			msgText := m.input.Value()
			if msgText != "" {
				// Append new message to history
				m.history += "\nYou: " + msgText
				m.messages.SetContent(m.history)
				m.messages.GotoBottom()
				m.input.Reset()
			}
			return m, nil
		}
	}

	// Update components
	m.contacts, cmd = m.contacts.Update(msg)
	cmds = append(cmds, cmd)
	m.messages, cmd = m.messages.Update(msg)
	cmds = append(cmds, cmd)
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) updateLayout() {
	inputHeight := 3
	padding := 1
	sidebarWidth := 30

	// Calculate total available height
	totalHeight := m.windowSize.Height - inputHeight - padding

	// Contact list
	m.contacts.SetSize(sidebarWidth, totalHeight)

	// Calculate content width (accounting for sidebar and padding)
	contentWidth := m.windowSize.Width - sidebarWidth - 4

	// Messages viewport
	m.messages.Width = contentWidth
	m.messages.Height = totalHeight - 2

	// Set input width to match content width (minus 2 for border padding)
	m.input.Width = contentWidth - 2
}

func (m model) View() string {
	sidebarWidth := 30
	contentWidth := m.windowSize.Width - sidebarWidth - 4

	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(m.windowSize.Height - 4).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.windowSize.Height - 7).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	inputBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(contentWidth)

	// Apply styles
	sidebar := sidebarStyle.Render(m.contacts.View())
	content := contentStyle.Render(m.messages.View())
	inputBox := inputBorder.Render(m.input.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		lipgloss.JoinVertical(
			lipgloss.Left,
			content,
			inputBox,
		),
	)
}
