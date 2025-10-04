package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	activeTab int
	width     int
	height    int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			m.activeTab = (m.activeTab + 1) % 2
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			m.activeTab = (m.activeTab - 1 + 2) % 2
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "loading..."
	}

	// Tabs
	doc := strings.Builder{}
	{
		var (
			tabs         []string
			tabContent   string
			inactiveTab  = lipgloss.NewStyle().Padding(0, 1)
			activeTab    = lipgloss.NewStyle().Inherit(inactiveTab).Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"})
			windowStyle  = lipgloss.NewStyle().BorderForeground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}).Padding(1, 0)
			tabsToRender = []string{"ANSI 16", "ANSI 256"}
		)

		for i, t := range tabsToRender {
			var style lipgloss.Style
			if i == m.activeTab {
				style = activeTab
			} else {
				style = inactiveTab
			}
			tabs = append(tabs, style.Render(t))
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
		doc.WriteString(row)
		doc.WriteString("\n")

		contentHeight := m.height - lipgloss.Height(row) - 2

		switch m.activeTab {
		case 0:
			tabContent = renderAnsi16(m.width, contentHeight)
		case 1:
			tabContent = renderAnsi256(m.width, contentHeight)
		}

		doc.WriteString(windowStyle.Width(m.width - windowStyle.GetHorizontalFrameSize()).Height(contentHeight).Render(tabContent))
	}
	return doc.String()
}

func renderAnsi16(width, height int) string {
	var s strings.Builder
	blockHeight := height / 16

	for i := 0; i < 16; i++ {
		color := fmt.Sprintf("%d", i)
		style := lipgloss.NewStyle().
			Height(blockHeight).
			Width(width - 5).
			Background(lipgloss.Color(color))

		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(4).Render(fmt.Sprintf("%3d:", i)),
			style.Render(""),
		))
		s.WriteString("\n")
	}
	return s.String()
}

func renderAnsi256(width, height int) string {
	var rows [][]string
	for i := 0; i < 16; i++ {
		var row []string
		for j := 0; j < 16; j++ {
			color := i*16 + j
			style := lipgloss.NewStyle().
				Width(width/16-4).
				Height(height/16).
				Align(lipgloss.Center, lipgloss.Center).
				Background(lipgloss.Color(fmt.Sprintf("%d", color)))

			box := lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().
					Width(4).
					Align(lipgloss.Right).
					Render(fmt.Sprintf("%d ", color)),
				style.Render(""),
			)

			row = append(row, box)
		}
		rows = append(rows, row)
	}

	var content []string
	for _, row := range rows {
		content = append(content, lipgloss.JoinHorizontal(lipgloss.Left, row...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
