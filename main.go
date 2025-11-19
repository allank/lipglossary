package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	activeTab int
	width     int
	height    int
	viewport  viewport.Model
	ready     bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-3) // -3 for tabs + border
			m.viewport.YPosition = 3
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 3
		}

		// Re-render content on resize
		m.viewport.SetContent(renderAnsi256(m.width))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			m.activeTab = (m.activeTab + 1) % 2
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			m.activeTab = (m.activeTab - 1 + 2) % 2
		case m.activeTab == 1 && key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
			m.viewport.ViewDown()
			return m, nil
		case m.activeTab == 1 && key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
			m.viewport.ViewUp()
			return m, nil
		}
	}

	// Handle viewport updates only if active tab is ANSI 256
	if m.activeTab == 1 {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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
			doc.WriteString(windowStyle.Width(m.width - windowStyle.GetHorizontalFrameSize()).Height(contentHeight).Render(tabContent))
		case 1:
			// Ensure content is set (in case of initial load or tab switch if we want to be safe, though resize handles it mostly)
			// But for static content that depends on width, we might want to update it if width changed?
			// Actually, Update handles SetContent on resize.
			// Just render the viewport.
			doc.WriteString(m.viewport.View())
		}
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

func renderAnsi256(width int) string {
	var s strings.Builder
	for i := 0; i < 256; i++ {
		color := fmt.Sprintf("%d", i)
		style := lipgloss.NewStyle().
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

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
