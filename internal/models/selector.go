package models

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ErrUserQuit = errors.New("user quit the application")

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type ModelSelector struct {
	list     list.Model
	choice   string
	quitting bool
}

func NewModelSelector(models []string) *ModelSelector {
	items := make([]list.Item, len(models))
	for i, model := range models {
		items[i] = item{
			title: model,
			desc:  fmt.Sprintf("Ollama model: %s", model),
		}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select an Ollama Model"

	return &ModelSelector{
		list: l,
	}
}

func (m *ModelSelector) Init() tea.Cmd {
	return nil
}

func (m *ModelSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.title
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ModelSelector) View() string {
	if m.choice != "" {
		return fmt.Sprintf("You chose %s\n", m.choice)
	}
	if m.quitting {
		return "Bye!\n"
	}
	return appStyle.Render(m.list.View())
}

var appStyle = lipgloss.NewStyle().Padding(1, 2)

func SelectModel(models []string) (string, error) {
	items := make([]list.Item, len(models))
	for i, model := range models {
		items[i] = item{title: model, desc: fmt.Sprintf("Ollama model: %s", model)}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select an Ollama Model"

	m := &ModelSelector{list: l}
	p := tea.NewProgram(m)

	model, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running model selector: %w", err)
	}

	if m, ok := model.(*ModelSelector); ok {
		if m.quitting {
			return "", ErrUserQuit
		}
		return m.choice, nil
	}

	return "", fmt.Errorf("could not get selected model")
}
