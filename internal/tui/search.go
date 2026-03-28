package tui

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/model"
)

type SearchResult struct {
	Request  *model.Request
	File     string
	MatchStr string
}

type SearchSelected struct {
	Request *model.Request
}

type SearchModel struct {
	input    string
	results  []SearchResult
	allItems []SearchResult
	cursor   int
	height   int
	width    int
}

func NewSearchModel() SearchModel {
	return SearchModel{}
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m *SearchModel) SetItems(items []SearchResult) {
	m.allItems = items
	m.results = items
	m.cursor = 0
}

func fuzzyMatch(query, target string) bool {
	query = strings.ToLower(query)
	target = strings.ToLower(target)
	qi := 0
	for ti := 0; ti < len(target) && qi < len(query); ti++ {
		if rune(query[qi]) == rune(target[ti]) || unicode.ToLower(rune(query[qi])) == unicode.ToLower(rune(target[ti])) {
			qi++
		}
	}
	return qi == len(query)
}

func (m SearchModel) filter(query string) []SearchResult {
	if query == "" {
		return m.allItems
	}
	var out []SearchResult
	for _, r := range m.allItems {
		haystack := fmt.Sprintf("%s %s %s", r.Request.Method, r.Request.URL, r.Request.Name)
		if fuzzyMatch(query, haystack) {
			out = append(out, r)
		}
	}
	return out
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
				m.results = m.filter(m.input)
				m.cursor = 0
			}
		case "j", "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor < len(m.results) {
				r := m.results[m.cursor]
				return m, func() tea.Msg { return SearchSelected{Request: r.Request} }
			}
		default:
			if k := msg.String(); len(k) == 1 {
				m.input += k
				m.results = m.filter(m.input)
				m.cursor = 0
			}
		}
	}
	return m, nil
}

func (m SearchModel) View() string {
	var sb strings.Builder
	inputLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4")).
		Render("/ " + m.input + "█")
	sb.WriteString(inputLine + "\n\n")

	for i, r := range m.results {
		line := fmt.Sprintf("%s  %s  (%s)",
			lipgloss.NewStyle().Foreground(methodColor(r.Request.Method)).Render(r.Request.Method),
			r.Request.URL,
			r.File)
		if i == m.cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#3D3D5C")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Render(line)
		}
		sb.WriteString(line + "\n")
	}
	if len(m.results) == 0 {
		sb.WriteString(dimStyle.Render("(no results)"))
	}
	return sb.String()
}
