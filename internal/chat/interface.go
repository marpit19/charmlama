package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marpit19/charmlama/internal/ollama"
)

type ChatInterface struct {
	model    string
	manager  *ollama.Manager
	viewport viewport.Model
	textarea textarea.Model
	messages []string
	err      error
	ready    bool
}

func NewChatInterface(model string, manager *ollama.Manager) *ChatInterface {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to CharmLlama! Start chatting with the AI.")

	return &ChatInterface{
		model:    model,
		manager:  manager,
		textarea: ta,
		viewport: vp,
		messages: []string{},
		ready:    false,
	}
}

func (c *ChatInterface) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, c.setReady)
}

func (c *ChatInterface) setReady() tea.Msg {
	return readyMsg{}
}

type readyMsg struct{}

func (c *ChatInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case readyMsg:
		c.ready = true
		return c, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return c, tea.Quit
		case tea.KeyEnter:
			if c.ready && c.textarea.Value() != "" {
				return c, c.sendMessage
			}
		}
	case tea.WindowSizeMsg:
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height - 3
		c.textarea.SetWidth(msg.Width)
		c.ready = true
	}

	c.textarea, cmd = c.textarea.Update(msg)
	return c, cmd
}

func (c *ChatInterface) View() string {
	if !c.ready {
		return "Initializing chat interface..."
	}
	return fmt.Sprintf(
		"%s\n\n%s",
		c.viewport.View(),
		c.textarea.View(),
	)
}

func (c *ChatInterface) sendMessage() tea.Msg {
	userMessage := c.textarea.Value()
	c.messages = append(c.messages, fmt.Sprintf("You: %s", userMessage))
	c.updateViewportContent()
	c.textarea.Reset()

	response, err := c.manager.SendMessage(c.model, userMessage)
	if err != nil {
		return errMsg{err}
	}

	c.messages = append(c.messages, fmt.Sprintf("AI: %s", response))
	c.updateViewportContent()
	return nil
}

func (c *ChatInterface) updateViewportContent() {
	c.viewport.SetContent(strings.Join(c.messages, "\n"))
	c.viewport.GotoBottom()
}

type errMsg struct{ error }

func (c *ChatInterface) Run() error {
	p := tea.NewProgram(c, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

var messageStyle = lipgloss.NewStyle().Padding(0, 1)
