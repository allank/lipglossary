// PROTOTYPE — throwaway UI exploration, not for production.
//
// Three structurally different layouts for browsing the ANSI 256 palette,
// switchable with tab / shift+tab / 1 / 2 / 3:
//  1. Mosaic  — a spatial 16x16 swatch grid with a detail inspector panel
//  2. Catalog — a bubbles/table data grid with a live preview card
//  3. Channels — a bubbles/list browser with bubbles/progress RGB meters
//
// Run with: go run . -prototype
//
// See docs/agents/issue-tracker.md before folding a winner into main.go —
// this file (and the -prototype flag hook in main.go) should be deleted
// once a variant is chosen.
package main

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const protoColorCount = 256

func protoColor(i int) (color.Color, int, int, int) {
	r, g, b := getRGB(i)
	return lipgloss.Color(fmt.Sprintf("%d", i)), r, g, b
}

func contrastFG(r, g, b int) color.Color {
	luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if luminance > 140 {
		return lipgloss.Color("0")
	}
	return lipgloss.Color("15")
}

type protoModel struct {
	width, height int
	variant       int
	ready         bool

	gridCursor int

	tbl table.Model

	lst  list.Model
	rBar progress.Model
	gBar progress.Model
	bBar progress.Model
}

func newProtoModel() protoModel {
	return protoModel{
		rBar: progress.New(progress.WithColors(lipgloss.Color("196")), progress.WithWidth(24)),
		gBar: progress.New(progress.WithColors(lipgloss.Color("46")), progress.WithWidth(24)),
		bBar: progress.New(progress.WithColors(lipgloss.Color("33")), progress.WithWidth(24)),
	}
}

func (m protoModel) Init() tea.Cmd {
	return nil
}

func protoTableColumns() []table.Column {
	return []table.Column{
		{Title: "#", Width: 4},
		{Title: "Swatch", Width: 8},
		{Title: "R", Width: 4},
		{Title: "G", Width: 4},
		{Title: "B", Width: 4},
	}
}

func protoTableRows() []table.Row {
	rows := make([]table.Row, protoColorCount)
	for i := 0; i < protoColorCount; i++ {
		color, r, g, b := protoColor(i)
		swatch := lipgloss.NewStyle().Background(color).Render("      ")
		rows[i] = table.Row{
			fmt.Sprintf("%3d", i),
			swatch,
			fmt.Sprintf("%3d", r),
			fmt.Sprintf("%3d", g),
			fmt.Sprintf("%3d", b),
		}
	}
	return rows
}

type swatchItem struct{ idx, r, g, b int }

func (s swatchItem) FilterValue() string { return fmt.Sprintf("%d", s.idx) }

type swatchDelegate struct{}

func (d swatchDelegate) Height() int  { return 1 }
func (d swatchDelegate) Spacing() int { return 0 }
func (d swatchDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}
func (d swatchDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(swatchItem)
	if !ok {
		return
	}
	color := lipgloss.Color(fmt.Sprintf("%d", it.idx))
	swatch := lipgloss.NewStyle().Background(color).Render("   ")
	label := fmt.Sprintf(" %3d  rgb(%3d,%3d,%3d)", it.idx, it.r, it.g, it.b)
	marker := "  "
	if index == m.Index() {
		marker = lipgloss.NewStyle().Bold(true).Render("> ")
	}
	fmt.Fprint(w, marker+swatch+label)
}

func protoListItems() []list.Item {
	items := make([]list.Item, protoColorCount)
	for i := 0; i < protoColorCount; i++ {
		_, r, g, b := protoColor(i)
		items[i] = swatchItem{idx: i, r: r, g: g, b: b}
	}
	return items
}

func (m protoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		footerHeight := 3
		contentHeight := msg.Height - footerHeight
		tableWidth := msg.Width * 2 / 3

		if !m.ready {
			m.tbl = table.New(
				table.WithColumns(protoTableColumns()),
				table.WithRows(protoTableRows()),
				table.WithFocused(true),
				table.WithHeight(contentHeight-2),
				table.WithWidth(tableWidth),
			)
			m.lst = list.New(protoListItems(), swatchDelegate{}, msg.Width*2/3, contentHeight)
			m.lst.Title = "256 Colors — Channel Explorer"
			m.lst.SetShowStatusBar(false)
			m.lst.SetShowHelp(false)
			m.ready = true
		} else {
			m.tbl.SetWidth(tableWidth)
			m.tbl.SetHeight(contentHeight - 2)
			m.lst.SetSize(msg.Width*2/3, contentHeight)
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Keystroke() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.variant = (m.variant + 1) % 3
			return m, nil
		case "shift+tab":
			m.variant = (m.variant + 2) % 3
			return m, nil
		case "1", "2", "3":
			m.variant = int(msg.Keystroke()[0] - '1')
			return m, nil
		}

		switch m.variant {
		case 0:
			switch msg.String() {
			case "up", "k":
				if m.gridCursor-16 >= 0 {
					m.gridCursor -= 16
				}
			case "down", "j":
				if m.gridCursor+16 < protoColorCount {
					m.gridCursor += 16
				}
			case "left", "h":
				if m.gridCursor%16 != 0 {
					m.gridCursor--
				}
			case "right", "l":
				if m.gridCursor%16 != 15 {
					m.gridCursor++
				}
			}
			return m, nil
		case 1:
			m.tbl, cmd = m.tbl.Update(msg)
			return m, cmd
		case 2:
			m.lst, cmd = m.lst.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m protoModel) viewMosaic() string {
	const cols, rows, cellW = 16, 16, 3

	var gridRows []string
	for row := 0; row < rows; row++ {
		var cells []string
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			color, r, g, b := protoColor(idx)
			content := "  "
			if idx == m.gridCursor {
				content = lipgloss.NewStyle().Foreground(contrastFG(r, g, b)).Bold(true).Render("><")
			}
			cells = append(cells, lipgloss.NewStyle().Background(color).Width(cellW).Render(content))
		}
		gridRows = append(gridRows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	grid := lipgloss.JoinVertical(lipgloss.Left, gridRows...)

	sel, r, g, b := protoColor(m.gridCursor)
	fg := contrastFG(r, g, b)
	swatch := lipgloss.NewStyle().Background(sel).Width(18).Height(4).Render("")

	info := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Swatch #%d", m.gridCursor)),
		fmt.Sprintf("RGB  %3d %3d %3d", r, g, b),
		"",
		swatch,
		"",
		lipgloss.NewStyle().Foreground(sel).Bold(true).Render("The quick brown fox"),
		lipgloss.NewStyle().Background(sel).Foreground(fg).Padding(0, 1).Render(" Sample Button "),
	)
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Render(info)

	return lipgloss.JoinHorizontal(lipgloss.Top, grid, "  ", panel)
}

func (m protoModel) viewCatalog() string {
	idx := m.tbl.Cursor()
	sel, r, g, b := protoColor(idx)
	fg := contrastFG(r, g, b)
	swatch := lipgloss.NewStyle().Background(sel).Width(20).Height(4).Render("")

	card := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Selected Color"),
		fmt.Sprintf("#%d", idx),
		fmt.Sprintf("RGB  %3d %3d %3d", r, g, b),
		"",
		swatch,
		"",
		lipgloss.NewStyle().Foreground(sel).Bold(true).Render("Aa The quick brown fox"),
		lipgloss.NewStyle().Background(sel).Foreground(fg).Padding(0, 1).Render(" Sample Button "),
	)
	preview := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).Padding(1, 2).Render(card)

	return lipgloss.JoinHorizontal(lipgloss.Top, m.tbl.View(), "  ", preview)
}

func (m protoModel) viewChannels() string {
	sel, ok := m.lst.SelectedItem().(swatchItem)
	if !ok {
		sel = swatchItem{}
	}
	color := lipgloss.Color(fmt.Sprintf("%d", sel.idx))
	swatch := lipgloss.NewStyle().Background(color).Width(20).Height(3).Render("")

	meters := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Swatch #%d", sel.idx)),
		"",
		"R "+m.rBar.ViewAs(float64(sel.r)/255),
		"G "+m.gBar.ViewAs(float64(sel.g)/255),
		"B "+m.bBar.ViewAs(float64(sel.b)/255),
		"",
		swatch,
	)
	panel := lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(1, 2).Render(meters)

	return lipgloss.JoinHorizontal(lipgloss.Top, m.lst.View(), "  ", panel)
}

var protoVariantNames = []string{
	"Mosaic — spatial grid + inspector",
	"Catalog — bubbles/table + preview card",
	"Channels — bubbles/list + bubbles/progress meters",
}

func (m protoModel) viewSwitcher() string {
	label := fmt.Sprintf(" PROTOTYPE  1:Mosaic 2:Catalog 3:Channels  (tab cycles)  |  now: %s  |  q quit ",
		protoVariantNames[m.variant])
	return lipgloss.NewStyle().
		Reverse(true).
		Bold(true).
		Width(m.width).
		Render(label)
}

func (m protoModel) View() tea.View {
	if !m.ready {
		return tea.NewView("loading...")
	}

	var body string
	switch m.variant {
	case 0:
		body = m.viewMosaic()
	case 1:
		body = m.viewCatalog()
	case 2:
		body = m.viewChannels()
	}

	v := tea.NewView(body + "\n\n" + m.viewSwitcher())
	v.AltScreen = true
	return v
}
