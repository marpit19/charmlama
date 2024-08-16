package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marpit19/charmlama/internal/ollama"
)

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF69B4")). // Hot Pink
			Bold(true)

	aiStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")). // Cyan
		Bold(true)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FF1493")). // Deep Pink
			Foreground(lipgloss.Color("#FFFFFF"))        // White text
)

type ChatInterface struct {
	model    string
	manager  *ollama.Manager
	messages []string
	viewport viewport.Model
	input    textinput.Model
	err      error
	quitting bool
	waiting  bool
	spinner  spinner.Model
	width    int
	height   int
}

func NewChatInterface(model string, manager *ollama.Manager) *ChatInterface {
	input := textinput.New()
	input.Placeholder = "Send a message... (Type /exit to quit)"
	input.Focus()

	vp := viewport.New(80, 20)
	vp.KeyMap.PageDown.SetEnabled(false)
	vp.KeyMap.PageUp.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))

	return &ChatInterface{
		model:    model,
		manager:  manager,
		messages: []string{},
		viewport: vp,
		input:    input,
		spinner:  s,
	}
}

func (c *ChatInterface) Init() tea.Cmd {
	return textinput.Blink
}

func (c *ChatInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "/exit":
			c.quitting = true
			return c, tea.Quit
		case "enter":
			if !c.waiting && c.input.Value() != "" {
				cmds = append(cmds, c.sendMessage)
			}
		case "up", "down":
			c.viewport, _ = c.viewport.Update(msg)
		}

	case tea.WindowSizeMsg:
		c.width, c.height = msg.Width, msg.Height
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height - 3
		c.input.Width = msg.Width - 4
		c.updateViewportContent()

	case userMessageMsg:
		c.addMessage("You", string(msg))
		c.waiting = true
		cmds = append(cmds, c.handleUserMessage(msg), c.spinner.Tick)

	case aiResponseMsg:
		c.waiting = false
		c.addMessage(c.model, string(msg))
	}

	if c.waiting {
		var cmd tea.Cmd
		c.spinner, cmd = c.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	c.input, _ = c.input.Update(msg)

	return c, tea.Batch(cmds...)
}

func (c *ChatInterface) View() string {
	var status string
	if c.waiting {
		status = fmt.Sprintf("%s AI is thinking...", c.spinner.View())
	} else {
		status = "Ready for your message"
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		c.viewport.View(),
		inputStyle.Render(c.input.View()),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#98FB98")).Render(status),
	)
}

func (c *ChatInterface) sendMessage() tea.Msg {
	userMessage := c.input.Value()
	c.input.SetValue("")
	return userMessageMsg(userMessage)
}

type (
	userMessageMsg string
	aiResponseMsg  string
)

func (c *ChatInterface) handleUserMessage(msg userMessageMsg) tea.Cmd {
	return func() tea.Msg {
		response, err := c.manager.SendMessage(c.model, string(msg))
		if err != nil {
			c.err = err
			return nil
		}
		return aiResponseMsg(response)
	}
}

func (c *ChatInterface) addMessage(sender, content string) {
	style := userStyle
	if sender != "You" {
		style = aiStyle
	}
	formattedMsg := style.Render(sender+":") + " " + content
	c.messages = append(c.messages, formattedMsg)
	c.updateViewportContent()
}

func (c *ChatInterface) updateViewportContent() {
	c.viewport.SetContent(strings.Join(c.messages, "\n\n"))
	c.viewport.GotoBottom()
}

func (c *ChatInterface) Run() (bool, error) {
	p := tea.NewProgram(c, tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return false, err
	}

	chatInterface, ok := m.(*ChatInterface)
	if !ok {
		return false, fmt.Errorf("could not assert type to *ChatInterface")
	}

	return chatInterface.quitting, chatInterface.err
}
