package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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

	activeInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#FF1493")). // Deep Pink
				Foreground(lipgloss.Color("#FFFFFF"))        // White text

	disabledInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#696969")). // Dim Gray
				Foreground(lipgloss.Color("#A9A9A9"))        // Dark Gray text

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98FB98"))
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
	renderer *glamour.TermRenderer
}

func NewChatInterface(model string, manager *ollama.Manager) *ChatInterface {
	input := textinput.New()
	input.Placeholder = "Send a message... (Type /exit to quit)"
	input.Focus()

	vp := viewport.New(80, 20)
	// vp.KeyMap.PageDown.SetEnabled(false)
	// vp.KeyMap.PageUp.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	return &ChatInterface{
		model:    model,
		manager:  manager,
		messages: []string{},
		viewport: vp,
		input:    input,
		spinner:  s,
		renderer: renderer,
	}
}

func (c *ChatInterface) Init() tea.Cmd {
	return textinput.Blink
}

func (c *ChatInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.waiting {
			// Ignore most key presses while waiting
			switch msg.String() {
			case "ctrl+c":
				c.quitting = true
				return c, tea.Quit
			default:
				return c, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "/exit":
			c.quitting = true
			return c, tea.Quit
		case "enter":
			if c.input.Value() == "/exit" {
				c.quitting = true
				return c, tea.Quit
			}
			if !c.waiting && c.input.Value() != "" {
				cmds = append(cmds, c.sendMessage)
			}
		case "up", "down", "pgup", "pgdown":
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
		c.input.Blur()
		cmds = append(cmds, c.handleUserMessage(msg), c.spinner.Tick)

	case aiResponseMsg:
		c.waiting = false
		c.addMessage(c.model, string(msg))
		c.input.Focus()
	}

	if c.waiting {
		var cmd tea.Cmd
		c.spinner, cmd = c.spinner.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		c.input, cmd = c.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// c.input, _ = c.input.Update(msg)

	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return c, tea.Batch(cmds...)
}

// func (c *ChatInterface) View() string {
// 	var status string
// 	var inputView string

// 	if c.waiting {
// 		status = fmt.Sprintf("%s AI is thinking...", c.spinner.View())
// 		inputView = disabledInputStyle.Render(c.input.View())
// 	} else {
// 		status = "Ready for your message"
// 		inputView = activeInputStyle.Render(c.input.View())
// 	}

// 	maxInputWidth := c.width - 4 // Adjust this value as needed
// 	if len(inputView) > maxInputWidth && maxInputWidth > 0 {
// 		inputView = inputView[:maxInputWidth] + "..."
// 	}

// 	return fmt.Sprintf(
// 		"%s\n%s\n%s",
// 		c.viewport.View(),
// 		inputView,
// 		lipgloss.NewStyle().Foreground(lipgloss.Color("#98FB98")).Render(status),
// 	)
// }

func (c *ChatInterface) View() string {
	var status string
	var inputView string

	if c.waiting {
		status = fmt.Sprintf("%s AI is thinking...", c.spinner.View())
		inputView = disabledInputStyle.Render(c.input.View())
	} else {
		status = "Ready for your message"
		inputView = activeInputStyle.Render(c.input.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		c.viewport.View(),
		inputView,
		statusStyle.Render(status),
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

// func (c *ChatInterface) addMessage(sender, content string) {
// 	style := userStyle
// 	if sender != "You" {
// 		style = aiStyle
// 	}
// 	formattedMsg := style.Render(sender+":") + " " + content
// 	wrappedMsg := c.wrapText(formattedMsg, c.width)
// 	c.messages = append(c.messages, wrappedMsg)
// 	c.updateViewportContent()
// }

func (c *ChatInterface) addMessage(sender, content string) {
	var formattedMsg string
	if sender == "You" {
		formattedMsg = userStyle.Render(sender+":") + " " + content
	} else {
		rendered, _ := c.renderer.Render(content)
		formattedMsg = aiStyle.Render(sender+":") + "\n" + rendered
	}
	c.messages = append(c.messages, formattedMsg)
	c.updateViewportContent()
}

func (c *ChatInterface) updateViewportContent() {
	content := strings.Join(c.messages, "\n\n")
	c.viewport.SetContent(content)
	c.viewport.GotoBottom()
}

// func (c *ChatInterface) wrapText(text string, width int) string {
// 	if width <= 0 {
// 		return text
// 	}
// 	words := strings.Fields(text)
// 	if len(words) == 0 {
// 		return text
// 	}

// 	var lines []string
// 	var currentLine string

// 	for _, word := range words {
// 		if len(currentLine)+len(word)+1 > width {
// 			lines = append(lines, strings.TrimSpace(currentLine))
// 			currentLine = word
// 		} else {
// 			if currentLine != "" {
// 				currentLine += " "
// 			}
// 			currentLine += word
// 		}
// 	}

// 	if currentLine != "" {
// 		lines = append(lines, strings.TrimSpace(currentLine))
// 	}

// 	return strings.Join(lines, "\n")
// }

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
