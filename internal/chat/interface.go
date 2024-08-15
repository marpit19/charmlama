package chat

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marpit19/charmlama/internal/ollama"
)

var (
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	aiStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

type message struct {
	sender  string
	content string
}

func (m message) Title() string       { return m.sender }
func (m message) Description() string { return m.content }
func (m message) FilterValue() string { return m.content }

type ChatInterface struct {
	model    string
	manager  *ollama.Manager
	messages list.Model
	textarea textarea.Model
	err      error
	quitting bool
}

func NewChatInterface(model string, manager *ollama.Manager) *ChatInterface {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	messageList := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	messageList.Title = "Chat with " + model
	messageList.SetShowStatusBar(false)
	messageList.SetFilteringEnabled(false)

	return &ChatInterface{
		model:    model,
		manager:  manager,
		messages: messageList,
		textarea: ta,
	}
}

func (c *ChatInterface) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, c.messages.StartSpinner())
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
				cmds = append(cmds, c.sendMessage)
			}
		}
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		c.messages.SetSize(msg.Width-h, msg.Height-v-3)
		c.textarea.SetWidth(msg.Width - h)
	case userMessageMsg:
		c.addMessage("You", string(msg))
		cmds = append(cmds, c.handleUserMessage(msg))
	case aiResponseMsg:
		c.addMessage("AI", string(msg))
	}

	var cmd tea.Cmd
	c.textarea, cmd = c.textarea.Update(msg)
	cmds = append(cmds, cmd)

	c.messages, cmd = c.messages.Update(msg)
	cmds = append(cmds, cmd)

	return c, tea.Batch(cmds...)
}

func (c *ChatInterface) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		c.messages.View(),
		c.textarea.View(),
	)
}

func (c *ChatInterface) sendMessage() tea.Msg {
	userMessage := c.textarea.Value()
	c.textarea.Reset()
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
	c.messages.InsertItem(len(c.messages.Items()), message{sender: sender, content: content})
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

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(message)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s: %s", i.sender, i.content)
	if i.sender == "You" {
		str = userStyle.Render(str)
	} else {
		str = aiStyle.Render(str)
	}
	fmt.Fprint(w, str)
}

var appStyle = lipgloss.NewStyle().Padding(1, 2)
