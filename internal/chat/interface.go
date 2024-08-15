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

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true)
	aiStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)

type ChatInterface struct {
	model    string
	manager  *ollama.Manager
	viewport viewport.Model
	textarea textarea.Model
	messages []string
	err      error
	ready    bool
	quitting bool
	debug    strings.Builder
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
	return textarea.Blink
}

func (c *ChatInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			c.quitting = true
			return c, tea.Quit
		case tea.KeyEnter:
			if c.textarea.Value() != "" {
				if c.textarea.Value() == "/exit" {
					c.quitting = true
					return c, tea.Quit
				}
				cmds = append(cmds, c.sendMessage)
			}
		}
	case tea.WindowSizeMsg:
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height - 3
		c.textarea.SetWidth(msg.Width)
		c.ready = true
	case userMessageMsg:
		c.log(fmt.Sprintf("Sending user message: %s", string(msg)))
		c.addMessage(fmt.Sprintf("You: %s", string(msg)))
		cmds = append(cmds, c.handleUserMessage(msg))
	case aiResponseMsg:
		c.log(fmt.Sprintf("Received AI response: %s", string(msg)))
		c.addMessage(fmt.Sprintf("AI: %s", string(msg)))
	case errMsg:
		c.log(fmt.Sprintf("Error occurred: %v", msg.err))
		c.err = msg
		return c, nil
	}

	var cmd tea.Cmd
	c.textarea, cmd = c.textarea.Update(msg)
	cmds = append(cmds, cmd)

	c.viewport, cmd = c.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return c, tea.Batch(cmds...)
}

func (c *ChatInterface) View() string {
	if !c.ready {
		return "Initializing chat interface..."
	}
	if c.err != nil {
		return fmt.Sprintf("Error: %v\n\nDebug info:\n%s", c.err, c.debug.String())
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\nType '/exit' to return to model selection",
		c.viewport.View(),
		c.textarea.View(),
	)
}

func (c *ChatInterface) sendMessage() tea.Msg {
	userMessage := c.textarea.Value()
	c.textarea.Reset()
	return userMessageMsg(userMessage)
}

type userMessageMsg string

type errMsg struct {
	err error
}

func (e errMsg) Error() string { return e.err.Error() }

func (c *ChatInterface) handleUserMessage(msg userMessageMsg) tea.Cmd {
	return func() tea.Msg {
		c.log("Sending message to Ollama...")
		response, err := c.manager.SendMessage(c.model, string(msg))
		if err != nil {
			c.log(fmt.Sprintf("Error from Ollama: %v", err))
			return errMsg{err}
		}
		c.log("Received response from Ollama")
		return aiResponseMsg(response)
	}
}

type aiResponseMsg string

func (c *ChatInterface) addMessage(message string) {
	c.messages = append(c.messages, message)
	c.updateViewportContent()
}

func (c *ChatInterface) updateViewportContent() {
	var content strings.Builder
	for _, msg := range c.messages {
		parts := strings.SplitN(msg, ": ", 2)
		if len(parts) == 2 {
			sender, message := parts[0], parts[1]
			var styledSender string
			if sender == "You" {
				styledSender = userStyle.Render(sender)
			} else {
				styledSender = aiStyle.Render(sender)
			}
			boxedMessage := boxStyle.Render(fmt.Sprintf("%s: %s", styledSender, message))
			content.WriteString(boxedMessage + "\n\n")
		}
	}
	c.viewport.SetContent(content.String())
	c.viewport.GotoBottom()
}

func (c *ChatInterface) log(message string) {
	c.debug.WriteString(message + "\n")
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

	return chatInterface.quitting, nil
}

var messageStyle = lipgloss.NewStyle().Padding(0, 1)
