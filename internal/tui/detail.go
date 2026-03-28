package tui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tidwall/pretty"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

type detailTab int

const (
	tabHeaders detailTab = iota
	tabBody
	tabTiming
)

type DetailModel struct {
	request        *model.Request
	response       *model.Response
	tab            detailTab
	sending        bool
	errMsg         string
	width          int
	height         int
	offset         int
	rootDir        string
	currentEnv     string
	envVars        map[string]string
	chainCtx       *parser.ChainContext
	cookies        *engine.CookieManager
	showHistory    bool
	historyEntries []history.HistoryEntry
	historyIdx     int
	diffMode       bool
	diffIdxA       int
}

type responseReceived struct {
	resp *model.Response
	err  error
}

type historyLoadedMsg struct {
	entries []history.HistoryEntry
}

func NewDetailModel(rootDir string, chainCtx *parser.ChainContext, cookies *engine.CookieManager) DetailModel {
	return DetailModel{
		rootDir:  rootDir,
		chainCtx: chainCtx,
		cookies:  cookies,
		envVars:  make(map[string]string),
	}
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case envVarsMsg:
		m.envVars = msg.vars
		m.currentEnv = msg.envName

	case RequestSelected:
		m.request = msg.Request
		m.response = nil
		m.offset = 0
		m.errMsg = ""
		m.showHistory = false

	case historyLoadedMsg:
		m.historyEntries = msg.entries
		m.historyIdx = 0
		m.diffMode = false

	case responseReceived:
		m.sending = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.response = msg.resp
			m.errMsg = ""
		}
		m.offset = 0

	case tea.KeyPressMsg:
		if m.showHistory {
			switch msg.String() {
			case "esc":
				m.showHistory = false
				m.diffMode = false
			case "j", "down":
				if m.historyIdx < len(m.historyEntries)-1 {
					m.historyIdx++
				}
			case "k", "up":
				if m.historyIdx > 0 {
					m.historyIdx--
				}
			case "enter":
				if m.historyIdx < len(m.historyEntries) {
					m.response = m.historyEntries[m.historyIdx].Response
					m.showHistory = false
					m.tab = tabBody
					m.offset = 0
				}
			case "d":
				if !m.diffMode {
					m.diffMode = true
					m.diffIdxA = m.historyIdx
				} else if m.historyIdx != m.diffIdxA &&
					m.diffIdxA < len(m.historyEntries) &&
					m.historyIdx < len(m.historyEntries) {
					a := &m.historyEntries[m.diffIdxA]
					b := &m.historyEntries[m.historyIdx]
					diffText := history.Diff(a, b)
					_ = diffText
					m.diffMode = false
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "1":
			m.tab = tabHeaders
			m.offset = 0
		case "2":
			m.tab = tabBody
			m.offset = 0
		case "3":
			m.tab = tabTiming
			m.offset = 0
		case "h":
			if m.request != nil {
				req := m.request
				rootDir := m.rootDir
				return m, func() tea.Msg {
					entries, _ := history.List(rootDir, req)
					return historyLoadedMsg{entries: entries}
				}
			}
			m.showHistory = !m.showHistory
		case "enter", "ctrl+r":
			if m.request != nil && !m.sending {
				m.sending = true
				m.errMsg = ""
				req := m.request
				envVars := m.envVars
				chainCtx := m.chainCtx
				cookies := m.cookies
				envName := m.currentEnv
				rootDir := m.rootDir
				return m, func() tea.Msg {
					resolved, _ := parser.ResolveRequest(req, envVars, chainCtx)
					loaded, err := parser.LoadFileBody(resolved, rootDir)
					if err != nil {
						loaded = resolved
					}
					jar := cookies.JarForEnv(envName)
					resp, err := engine.ExecuteWithJar(loaded, jar)
					return responseReceived{resp: resp, err: err}
				}
			}
		case "j", "down":
			m.offset++
		case "k", "up":
			if m.offset > 0 {
				m.offset--
			}
		}
	}
	return m, nil
}

func (m DetailModel) View() string {
	if m.showHistory {
		return m.historyView()
	}

	if m.request == nil {
		return dimStyle.Render("Request / Response\n\n(select a request to view)")
	}

	var sb strings.Builder

	if m.sending {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render("Sending..."))
		sb.WriteString("\n\n")
	}

	if m.errMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F44336")).Render("Error: " + m.errMsg))
		sb.WriteString("\n\n")
		sb.WriteString(requestView(m.request))
		return sb.String()
	}

	if m.response == nil {
		sb.WriteString(requestView(m.request))
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Enter / ctrl+r to send  │  h: history"))
		return sb.String()
	}

	tabs := []string{"1:Headers", "2:Body", "3:Timing"}
	var tabBar strings.Builder
	for i, t := range tabs {
		if detailTab(i) == m.tab {
			tabBar.WriteString(lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("#CDD6F4")).Render(t))
		} else {
			tabBar.WriteString(dimStyle.Render(t))
		}
		if i < len(tabs)-1 {
			tabBar.WriteString("  ")
		}
	}
	sb.WriteString(statusLine(m.response))
	sb.WriteString("\n")
	sb.WriteString(tabBar.String())
	sb.WriteString("\n\n")

	var body string
	switch m.tab {
	case tabHeaders:
		body = headersView(m.response)
	case tabBody:
		body = bodyView(m.response)
	case tabTiming:
		body = timingView(m.response)
	}

	lines := strings.Split(body, "\n")
	end := m.offset + m.height - 5
	if end > len(lines) {
		end = len(lines)
	}
	start := m.offset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	for _, l := range lines[start:end] {
		sb.WriteString(l + "\n")
	}

	return sb.String()
}

func (m DetailModel) historyView() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")).Bold(true).Render("Response History"))
	sb.WriteString("\n\n")
	if len(m.historyEntries) == 0 {
		sb.WriteString(dimStyle.Render("(no history for this request)"))
		return sb.String()
	}
	for i, e := range m.historyEntries {
		ts := e.Timestamp.Format("2006-01-02 15:04:05")
		status := ""
		if e.Response != nil {
			status = fmt.Sprintf(" %d", e.Response.StatusCode)
		}
		line := fmt.Sprintf("%s%s  [%s]", ts, status, e.Environment)
		if m.diffMode && i == m.diffIdxA {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9800")).Render("A " + line)
		} else if i == m.historyIdx {
			line = lipgloss.NewStyle().Background(lipgloss.Color("#3D3D5C")).Foreground(lipgloss.Color("#FFFFFF")).Render(line)
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n")
	if m.diffMode {
		sb.WriteString(dimStyle.Render("Select second entry and press d to diff  │  esc: cancel"))
	} else {
		sb.WriteString(dimStyle.Render("Enter: view  │  d: diff  │  esc: close"))
	}
	return sb.String()
}

func requestView(req *model.Request) string {
	var sb strings.Builder
	method := lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50")).Bold(true).Render(req.Method)
	sb.WriteString(fmt.Sprintf("%s %s\n\n", method, req.URL))
	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB")).Render(h.Key),
			h.Value))
	}
	if req.Body != "" {
		sb.WriteString("\n" + req.Body)
	}
	return sb.String()
}

func statusLine(resp *model.Response) string {
	code := resp.StatusCode
	var clr color.Color
	switch {
	case code >= 200 && code < 300:
		clr = lipgloss.Color("#4CAF50")
	case code >= 300 && code < 400:
		clr = lipgloss.Color("#FFFF00")
	case code >= 400 && code < 500:
		clr = lipgloss.Color("#FF9800")
	default:
		clr = lipgloss.Color("#F44336")
	}
	return lipgloss.NewStyle().Foreground(clr).Bold(true).Render(fmt.Sprintf("HTTP %d %s", code, resp.Status))
}

func headersView(resp *model.Response) string {
	var sb strings.Builder
	for _, h := range resp.Headers {
		key := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB")).Render(h.Key)
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, h.Value))
	}
	return sb.String()
}

func bodyView(resp *model.Response) string {
	if len(resp.Body) == 0 {
		return dimStyle.Render("(empty body)")
	}
	ct := strings.ToLower(resp.ContentType)
	if strings.Contains(ct, "json") {
		return string(pretty.Color(pretty.Pretty(resp.Body), nil))
	}
	return string(resp.Body)
}

func timingView(resp *model.Response) string {
	t := resp.Timing
	total := t.Total.Milliseconds()
	if total == 0 {
		return dimStyle.Render("(no timing data)")
	}

	barWidth := 24
	phases := []struct {
		name string
		ms   int64
	}{
		{"DNS    ", t.DNS.Milliseconds()},
		{"Connect", t.Connect.Milliseconds()},
		{"TLS    ", t.TLS.Milliseconds()},
		{"TTFB   ", t.TTFB.Milliseconds()},
		{"Body   ", t.BodyRead.Milliseconds()},
		{"Total  ", total},
	}

	var sb strings.Builder
	for _, p := range phases {
		filled := int(int64(barWidth) * p.ms / total)
		if p.ms > 0 && filled == 0 {
			filled = 1
		}
		empty := barWidth - filled
		if empty < 0 {
			empty = 0
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
		sb.WriteString(fmt.Sprintf("%s  %s  %dms\n", p.name, bar, p.ms))
	}
	return sb.String()
}
