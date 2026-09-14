package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type providerOption struct {
	label    string
	provider string
}

var providerOptions = []providerOption{
	{label: "ChatGPT", provider: "chatgpt"},
	{label: "Claude", provider: "claude"},
	{label: "Gemini", provider: "gemini"},
	{label: "Kimi", provider: "kimi"},
	{label: "Custom URL", provider: "custom"},
}

type providerSelection struct {
	Provider  string
	CustomURL string
}

type providerPickerModel struct {
	current        string
	cursor         int
	enteringCustom bool
	customURL      string
	selection      providerSelection
	cancelled      bool
}

func newProviderPicker(current string, customURL ...string) providerPickerModel {
	model := providerPickerModel{current: current}
	if current == "custom" && len(customURL) > 0 {
		model.customURL = customURL[0]
	}
	for index, option := range providerOptions {
		if option.provider == current {
			model.cursor = index
			break
		}
	}
	return model
}

func (providerPickerModel) Init() tea.Cmd { return nil }

func (m providerPickerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.enteringCustom {
		return m.updateCustomURL(key)
	}

	switch key.String() {
	case "ctrl+c", "esc", "q":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		m.cursor = (m.cursor - 1 + len(providerOptions)) % len(providerOptions)
	case "down", "j":
		m.cursor = (m.cursor + 1) % len(providerOptions)
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(providerOptions) - 1
	case "enter":
		provider := providerOptions[m.cursor].provider
		if provider == "custom" {
			m.enteringCustom = true
			return m, nil
		}
		m.selection.Provider = provider
		return m, tea.Quit
	}
	return m, nil
}

func (m providerPickerModel) updateCustomURL(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyEsc:
		m.enteringCustom = false
		return m, nil
	case tea.KeyEnter:
		m.selection = providerSelection{Provider: "custom", CustomURL: strings.TrimSpace(m.customURL)}
		return m, tea.Quit
	case tea.KeyBackspace, tea.KeyDelete:
		runes := []rune(m.customURL)
		if len(runes) > 0 {
			m.customURL = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.customURL += string(key.Runes)
	case tea.KeySpace:
		m.customURL += " "
	}
	return m, nil
}

func (m providerPickerModel) View() string {
	if m.enteringCustom {
		return fmt.Sprintf(
			"Choose AI provider\n\nCustom HTTPS URL\n> %s\n\nenter save • esc back • ctrl+c cancel\n",
			m.customURL,
		)
	}

	var view strings.Builder
	fmt.Fprintf(&view, "Choose AI provider\nCurrent: %s\n\n", m.current)
	for index, option := range providerOptions {
		cursor := "  "
		if index == m.cursor {
			cursor = "> "
		}
		current := ""
		if option.provider == m.current {
			current = " (current)"
		}
		fmt.Fprintf(&view, "%s%s%s\n", cursor, option.label, current)
	}
	view.WriteString("\n↑/k up • ↓/j down • enter select • esc/q cancel\n")
	return view.String()
}
