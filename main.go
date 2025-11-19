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
	activeTab  int
	width      int
	height     int
	viewport   viewport.Model
	ready      bool
	rThreshold int
	gThreshold int
	bThreshold int
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

		// Calculate available height for viewport
		// Header (Tabs) + Thresholds + Status Bar + Margins
		// Tabs: 3 lines (border + text) approx? Let's say 2 for now + 1 for gap filling
		// Thresholds: 1 line
		// Status Bar: 1 line
		// Total deduction: ~5-6 lines. Let's calculate dynamically in View, but here we need an estimate or exact.
		// Let's assume:
		// Tabs: 2 lines (1 text + 1 border)
		// Thresholds: 1 line
		// Status Bar: 1 line
		// Total: 4 lines overhead.
		verticalMargins := 4

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargins)
			m.viewport.YPosition = verticalMargins // Approximate
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}

		// Re-render content on resize
		m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))

	case tea.KeyMsg:
		step := 17
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			m.activeTab = (m.activeTab + 1) % 2
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			m.activeTab = (m.activeTab - 1 + 2) % 2
		// Threshold controls
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.rThreshold = clamp(m.rThreshold+step, 0, 255)
			m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))
		case key.Matches(msg, key.NewBinding(key.WithKeys("R"))):
			m.rThreshold = clamp(m.rThreshold-step, 0, 255)
			m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.gThreshold = clamp(m.gThreshold+step, 0, 255)
			m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.gThreshold = clamp(m.gThreshold-step, 0, 255)
			m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))
		case key.Matches(msg, key.NewBinding(key.WithKeys("b"))):
			m.bThreshold = clamp(m.bThreshold+step, 0, 255)
			m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))
		case key.Matches(msg, key.NewBinding(key.WithKeys("B"))):
			m.bThreshold = clamp(m.bThreshold-step, 0, 255)
			m.viewport.SetContent(renderAnsi256(m.width, m.viewport.Height, m.rThreshold, m.gThreshold, m.bThreshold))
		}
	}

	// Handle viewport updates only if active tab is ANSI 256
	if m.activeTab == 1 {
		switch msg := msg.(type) { // Re-evaluate msg for viewport specific keys
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
				m.viewport.ViewDown()
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
				m.viewport.ViewUp()
				return m, nil
			}
		}
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (m model) View() string {
	if m.width == 0 {
		return "loading..."
	}

	var (
		// Styles
		highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
		inactive  = lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#505050"}

		activeTabBorder = lipgloss.Border{
			Top:         "─",
			Bottom:      " ",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┘",
			BottomRight: "└",
		}

		tabBorder = lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┴",
			BottomRight: "┴",
		}

		tab = lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(inactive).
			Padding(0, 1)

		activeTab = tab.
				Border(activeTabBorder, true).
				Bold(true).
				Foreground(highlight)

		tabGap = lipgloss.NewStyle().
			Border(lipgloss.Border{
				Bottom: "─",
			}).
			BorderForeground(highlight)

		// Status Bar
		statusBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
				Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"}).
				Padding(0, 1)

		statusText = lipgloss.NewStyle().Inherit(statusBarStyle)

		statusKey = lipgloss.NewStyle().
				Inherit(statusBarStyle).
				Foreground(lipgloss.Color("#FFFDF5")).
				Background(lipgloss.Color("#FF5F87")).
				Padding(0, 1).
				MarginRight(1)
	)

	// Tabs
	var tabs []string
	tabsToRender := []string{"ANSI 16", "ANSI 256"}

	for i, t := range tabsToRender {
		if i == m.activeTab {
			tabs = append(tabs, activeTab.Render(t))
		} else {
			tabs = append(tabs, tab.Render(t))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	gap := tabGap.Width(max(0, m.width-lipgloss.Width(row)-2)).Render("")
	header := lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)

	// Thresholds
	thresholds := lipgloss.JoinHorizontal(lipgloss.Center,
		fmt.Sprintf("Filters (R/G/B): %d/%d/%d", m.rThreshold, m.gThreshold, m.bThreshold),
	)

	// Status Bar
	statusBar := statusBarStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			statusKey.Render("h/l"), statusText.Render("tabs"),
			statusKey.Render("j/k"), statusText.Render("scroll"),
			statusKey.Render("^d/^u"), statusText.Render("page"),
			statusKey.Render("r/g/b"), statusText.Render("filter"),
			statusKey.Render("q"), statusText.Render("quit"),
		),
	)

	// Content
	// Calculate available height
	// Header height + Thresholds height + Status Bar height
	headerHeight := lipgloss.Height(header)
	thresholdsHeight := lipgloss.Height(thresholds)
	statusBarHeight := lipgloss.Height(statusBar)

	contentHeight := m.height - headerHeight - thresholdsHeight - statusBarHeight

	// Ensure viewport height is updated if it doesn't match
	if m.viewport.Height != contentHeight && contentHeight > 0 {
		m.viewport.Height = contentHeight
	}

	doc := strings.Builder{}
	doc.WriteString(header)
	doc.WriteString("\n")
	doc.WriteString(thresholds)
	doc.WriteString("\n")

	var tabContent string
	switch m.activeTab {
	case 0:
		tabContent = renderAnsi16(m.width, contentHeight, m.rThreshold, m.gThreshold, m.bThreshold)
		doc.WriteString(tabContent) // renderAnsi16 handles its own sizing mostly, but we pass contentHeight
	case 1:
		doc.WriteString(m.viewport.View())
	}

	// Fill remaining vertical space if any (for ANSI 16 which might be short)
	// Actually ANSI 16 fills height passed to it.

	// Status Bar at bottom
	// We need to ensure the content pushes the status bar to the bottom or we just append it.
	// Since we calculated contentHeight to fill the space, appending should place it at the bottom.
	// However, if content is short (ANSI 16), we might need to pad?
	// renderAnsi16 uses blockHeight = height / 16. It tries to fill.

	// If we are in viewport mode, viewport fills height.

	doc.WriteString("\n")
	doc.WriteString(statusBar)

	return doc.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getRGB(c int) (r, g, b int) {
	if c < 16 {
		// Standard ANSI colors (approximate values)
		// 0-7: Standard, 8-15: High Intensity
		// Using standard VGA colors
		palette := [][3]int{
			{0, 0, 0}, {170, 0, 0}, {0, 170, 0}, {170, 85, 0},
			{0, 0, 170}, {170, 0, 170}, {0, 170, 170}, {170, 170, 170},
			{85, 85, 85}, {255, 85, 85}, {85, 255, 85}, {255, 255, 85},
			{85, 85, 255}, {255, 85, 255}, {85, 255, 255}, {255, 255, 255},
		}
		return palette[c][0], palette[c][1], palette[c][2]
	}

	if c < 232 {
		// 6x6x6 Color Cube
		// 16 + 36*r + 6*g + b
		c -= 16
		bVal := c % 6
		gVal := (c / 6) % 6
		rVal := c / 36

		vals := []int{0, 95, 135, 175, 215, 255}
		return vals[rVal], vals[gVal], vals[bVal]
	}

	// Grayscale 232-255
	// 232 is 8, 255 is 238. Step is 10.
	val := 8 + (c-232)*10
	return val, val, val
}

func renderAnsi16(width, height, rThresh, gThresh, bThresh int) string {
	var s strings.Builder
	blockHeight := height / 16
	if blockHeight < 1 {
		blockHeight = 1
	}

	for i := 0; i < 16; i++ {
		r, g, b := getRGB(i)
		if r < rThresh || g < gThresh || b < bThresh {
			continue
		}

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

func renderAnsi256(width, height, rThresh, gThresh, bThresh int) string {
	var s strings.Builder
	blockHeight := height / 16
	if blockHeight < 1 {
		blockHeight = 1
	}

	for i := 0; i < 256; i++ {
		r, g, b := getRGB(i)
		if r < rThresh || g < gThresh || b < bThresh {
			continue
		}

		color := fmt.Sprintf("%d", i)
		style := lipgloss.NewStyle().
			Width(width - 5).
			Height(blockHeight).
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
