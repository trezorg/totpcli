package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search"
	"github.com/trezorg/totpcli/internal/totpcode"
)

type Generator interface {
	Generate(provider.Provider, time.Time) (totpcode.Result, error)
}

type Model struct {
	providers []provider.Provider
	engine    search.Engine
	input     textinput.Model
	filtered  []provider.Provider
	cursor    int
	selected  *provider.Provider
	code      string
	remaining int
	err       error
	generator Generator
	now       func() time.Time
}

type tickMsg time.Time

type styleSchema struct {
	search           lipgloss.Style
	providerRow      lipgloss.Style
	selectedProvider lipgloss.Style
	codeLabel        lipgloss.Style
	codeValue        lipgloss.Style
	remaining        lipgloss.Style
	empty            lipgloss.Style
	help             lipgloss.Style
	err              lipgloss.Style
}

var styles = styleSchema{
	search:           lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	providerRow:      lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	selectedProvider: lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Bold(true),
	codeLabel:        lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	codeValue:        lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
	remaining:        lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
	empty:            lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
	help:             lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	err:              lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
}

func NewModel(providers []provider.Provider, engine search.Engine, initialSearch string, generator Generator, now func() time.Time) Model {
	input := textinput.New()
	input.Placeholder = "Search providers"
	input.Prompt = "Search: "
	input.SetValue(initialSearch)
	input.Focus()
	input.CharLimit = 256
	input.Width = 40

	if now == nil {
		now = time.Now
	}

	model := Model{
		providers: append([]provider.Provider(nil), providers...),
		engine:    engine,
		input:     input,
		generator: generator,
		now:       now,
	}
	model.applySearch()
	return model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.MoveUp()
			return m, nil
		case "down", "j":
			m.MoveDown()
			return m, nil
		case "enter":
			m.Select()
			return m, nil
		}
	case tickMsg:
		m.Refresh()
		return m, tick()
	}

	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.applySearch()
	}
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(styles.search.Render(m.input.View()))
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(styles.empty.Render("No providers found."))
		b.WriteString("\n")
	} else {
		for i, candidate := range m.filtered {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			row := fmt.Sprintf("%s %s", cursor, candidate.DisplayLabel())
			if i == m.cursor {
				b.WriteString(styles.selectedProvider.Render(row))
			} else {
				b.WriteString(styles.providerRow.Render(row))
			}
			b.WriteString("\n")
		}
	}

	if m.err != nil {
		fmt.Fprintf(&b, "\n%s\n", styles.err.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.code != "" && m.selected != nil {
		fmt.Fprintf(&b, "\n%s\n", styles.selectedProvider.Render(m.selected.DisplayLabel()))
		fmt.Fprintf(&b, "%s %s\n", styles.codeLabel.Render("Code:"), styles.codeValue.Render(m.code))
		fmt.Fprintf(&b, "%s\n", styles.remaining.Render(fmt.Sprintf("Remaining: %ds", m.remaining)))
	}

	b.WriteString("\n")
	b.WriteString(styles.help.Render("Enter selects. q quits."))
	b.WriteString("\n")
	return b.String()
}

func (m *Model) SetSearch(query string) {
	m.input.SetValue(query)
	m.applySearch()
}

func (m Model) SearchValue() string {
	return m.input.Value()
}

func (m Model) Filtered() []provider.Provider {
	return append([]provider.Provider(nil), m.filtered...)
}

func (m *Model) MoveDown() {
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
	}
}

func (m *Model) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) Select() bool {
	if len(m.filtered) == 0 {
		return false
	}

	selected := m.filtered[m.cursor]
	m.selected = &selected
	m.Refresh()
	return true
}

func (m *Model) Refresh() {
	if m.selected == nil || m.generator == nil {
		return
	}

	result, err := m.generator.Generate(*m.selected, m.now())
	if err != nil {
		m.code = ""
		m.remaining = 0
		m.err = err
		return
	}

	m.err = nil
	m.code = result.Code
	m.remaining = result.Remaining
}

func (m Model) Selected() provider.Provider {
	if m.selected == nil {
		return provider.Provider{}
	}
	return *m.selected
}

func (m Model) Code() string {
	return m.code
}

func (m Model) Remaining() int {
	return m.remaining
}

func (m Model) Err() error {
	return m.err
}

func (m *Model) applySearch() {
	if m.engine == nil {
		m.engine = search.NewLexical(m.providers)
	}
	m.filtered = m.engine.Search(m.input.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
