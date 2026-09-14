package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	accentColor = lipgloss.AdaptiveColor{Light: "#5A38B5", Dark: "#C4A7FF"}
	mutedColor  = lipgloss.AdaptiveColor{Light: "#6B6673", Dark: "#928B9D"}
	textColor   = lipgloss.AdaptiveColor{Light: "#221D29", Dark: "#F3EDF7"}

	pickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)
	pickerCurrentStyle = lipgloss.NewStyle().
				Foreground(mutedColor)
	pickerPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Padding(0, 1).
				Width(32)
	pickerRowStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Padding(0, 1).
			Width(28)
	pickerSelectedStyle = pickerRowStyle.Copy().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#21182B"}).
				Background(accentColor)
	pickerBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)
	pickerHelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
	pickerInputStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Padding(0, 1).
				Width(42)
	pickerCursorStyle = lipgloss.NewStyle().
				Foreground(accentColor)
	pickerContainerStyle = lipgloss.NewStyle().
				Margin(1, 2)
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
		input := pickerInputStyle.Render(m.customURL + pickerCursorStyle.Render("█"))
		content := strings.Join([]string{
			pickerTitleStyle.Render("Choose AI provider"),
			pickerCurrentStyle.Render("Custom HTTPS URL"),
			input,
			pickerHelpStyle.Render("enter save  •  esc back  •  ctrl+c cancel"),
		}, "\n\n")
		return pickerContainerStyle.Render(content) + "\n"
	}

	var rows strings.Builder
	for index, option := range providerOptions {
		cursor := "  "
		if index == m.cursor {
			cursor = "› "
		}
		current := ""
		if option.provider == m.current {
			current = "  ● current"
			if index != m.cursor {
				current = "  " + pickerBadgeStyle.Render("● current")
			}
		}
		row := cursor + option.label + current
		if index == m.cursor {
			row = pickerSelectedStyle.Render(row)
		} else {
			row = pickerRowStyle.Render(row)
		}
		rows.WriteString(row)
		rows.WriteByte('\n')
	}

	content := strings.Join([]string{
		pickerTitleStyle.Render("Choose AI provider"),
		pickerCurrentStyle.Render(fmt.Sprintf("Current: %s", m.current)),
		pickerPanelStyle.Render(strings.TrimSuffix(rows.String(), "\n")),
		pickerHelpStyle.Render("↑/k up  •  ↓/j down  •  enter select  •  esc/q cancel"),
	}, "\n")
	return pickerContainerStyle.Render(content) + "\n"
}
