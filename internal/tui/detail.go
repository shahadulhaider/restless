package tui

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"strings"
	"unicode/utf8"

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

	// Body viewer enhancements
	wordWrap     bool
	showLineNums bool
	searching    bool   // true when search input is active
	searchQuery  string // current search text
	searchHits   []int  // line indices that match
	searchIdx    int    // current match index in searchHits
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
		rootDir:      rootDir,
		chainCtx:     chainCtx,
		cookies:      cookies,
		envVars:      make(map[string]string),
		showLineNums: true,
	}
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

// viewableHeight returns how many body lines fit in the detail pane.
func (m DetailModel) viewableHeight() int {
	h := m.height - 6 // status line + tabs + padding
	if m.searching {
		h-- // search bar takes a line
	}
	if h < 1 {
		h = 1
	}
	return h
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
		m.clearSearch()

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
		m.clearSearch()

	case tea.KeyPressMsg:
		if m.showHistory {
			return m.updateHistory(msg)
		}
		if m.searching {
			return m.updateSearch(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m DetailModel) updateHistory(msg tea.KeyPressMsg) (DetailModel, tea.Cmd) {
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
			_ = history.Diff(a, b)
			m.diffMode = false
		}
	}
	return m, nil
}

func (m DetailModel) updateSearch(msg tea.KeyPressMsg) (DetailModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
	case "enter":
		m.searching = false
		// Keep search results visible, jump to first match
		if len(m.searchHits) > 0 {
			m.offset = m.searchHits[m.searchIdx]
		}
	case "backspace":
		if len(m.searchQuery) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.searchQuery)
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-size]
			m.rebuildSearchHits()
		}
	default:
		if k := msg.String(); len([]rune(k)) == 1 {
			m.searchQuery += k
			m.rebuildSearchHits()
		}
	}
	return m, nil
}

func (m DetailModel) updateNormal(msg tea.KeyPressMsg) (DetailModel, tea.Cmd) {
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

	// --- Scrolling ---
	case "j", "down":
		m.offset++
	case "k", "up":
		if m.offset > 0 {
			m.offset--
		}
	case "ctrl+d":
		m.offset += m.viewableHeight() / 2
	case "ctrl+u":
		m.offset -= m.viewableHeight() / 2
		if m.offset < 0 {
			m.offset = 0
		}
	case "g":
		m.offset = 0
	case "G":
		// Jump to bottom — clamped in View()
		m.offset = 999999

	// --- Body viewer controls ---
	case "w":
		m.wordWrap = !m.wordWrap
		m.offset = 0
	case "l":
		m.showLineNums = !m.showLineNums
	case "f":
		if m.response != nil && m.tab == tabBody {
			m.searching = true
			m.searchQuery = ""
			m.searchHits = nil
			m.searchIdx = 0
		}
	case "n":
		// Next search match
		if len(m.searchHits) > 0 {
			m.searchIdx = (m.searchIdx + 1) % len(m.searchHits)
			m.offset = m.searchHits[m.searchIdx]
		}
	case "N":
		// Previous search match
		if len(m.searchHits) > 0 {
			m.searchIdx = (m.searchIdx - 1 + len(m.searchHits)) % len(m.searchHits)
			m.offset = m.searchHits[m.searchIdx]
		}
	}
	return m, nil
}

// --- Search helpers ---

func (m *DetailModel) clearSearch() {
	m.searching = false
	m.searchQuery = ""
	m.searchHits = nil
	m.searchIdx = 0
}

func (m *DetailModel) rebuildSearchHits() {
	m.searchHits = nil
	m.searchIdx = 0
	if m.searchQuery == "" || m.response == nil {
		return
	}
	body := m.getBodyText()
	lines := strings.Split(body, "\n")
	query := strings.ToLower(m.searchQuery)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.searchHits = append(m.searchHits, i)
		}
	}
}

// --- View ---

func (m DetailModel) View() string {
	if m.showHistory {
		return m.historyView()
	}

	if m.request == nil {
		return dimStyle.Render("Request / Response\n\n(select a request to view)")
	}

	var sb strings.Builder

	if m.sending {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render("⏳ Sending..."))
		sb.WriteString("\n\n")
	}

	if m.errMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F44336")).Render("✗ " + m.errMsg))
		sb.WriteString("\n\n")
		sb.WriteString(requestView(m.request))
		return sb.String()
	}

	if m.response == nil {
		sb.WriteString(requestView(m.request))
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Enter: send  │  h: history"))
		return sb.String()
	}

	// Status line + body info
	sb.WriteString(statusLine(m.response))
	sb.WriteString("  ")
	sb.WriteString(dimStyle.Render(bodyInfo(m.response)))
	sb.WriteString("\n")

	// Tab bar
	sb.WriteString(m.tabBar())
	sb.WriteString("\n\n")

	// Tab content
	var body string
	switch m.tab {
	case tabHeaders:
		body = headersView(m.response)
	case tabBody:
		body = m.enhancedBodyView()
	case tabTiming:
		body = timingView(m.response)
	}

	lines := strings.Split(body, "\n")

	// Clamp offset
	maxOffset := len(lines) - m.viewableHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}

	end := m.offset + m.viewableHeight()
	if end > len(lines) {
		end = len(lines)
	}
	for _, l := range lines[m.offset:end] {
		sb.WriteString(l + "\n")
	}

	// Scroll indicator
	if len(lines) > m.viewableHeight() {
		pct := 0
		if maxOffset > 0 {
			pct = m.offset * 100 / maxOffset
		}
		sb.WriteString(dimStyle.Render(fmt.Sprintf("── %d%% (%d/%d lines) ──", pct, m.offset+1, len(lines))))
	}

	// Search bar
	if m.searching {
		sb.WriteString("\n")
		matchInfo := ""
		if m.searchQuery != "" {
			matchInfo = fmt.Sprintf(" [%d/%d]", m.searchIdx+1, len(m.searchHits))
			if len(m.searchHits) == 0 {
				matchInfo = " [no match]"
			}
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(colorBorderActive).Render("find: " + m.searchQuery + "█" + matchInfo))
	}

	return sb.String()
}

func (m DetailModel) tabBar() string {
	tabs := []string{"1:Headers", "2:Body", "3:Timing"}
	var sb strings.Builder
	for i, t := range tabs {
		if detailTab(i) == m.tab {
			sb.WriteString(lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("#CDD6F4")).Render(t))
		} else {
			sb.WriteString(dimStyle.Render(t))
		}
		if i < len(tabs)-1 {
			sb.WriteString("  ")
		}
	}
	// Body tab extras
	if m.tab == tabBody && m.response != nil {
		extras := []string{}
		if m.wordWrap {
			extras = append(extras, "wrap:on")
		}
		if m.searchQuery != "" && !m.searching {
			extras = append(extras, fmt.Sprintf("/%s", m.searchQuery))
		}
		if len(extras) > 0 {
			sb.WriteString("  " + dimStyle.Render(strings.Join(extras, " │ ")))
		}
	}
	return sb.String()
}

// getBodyText returns the formatted body text (without line numbers or highlights).
func (m DetailModel) getBodyText() string {
	if m.response == nil || len(m.response.Body) == 0 {
		return ""
	}
	return formatBody(m.response, m.width-8) // leave room for line numbers
}

// enhancedBodyView renders the body with line numbers, search highlights, and word wrap.
func (m DetailModel) enhancedBodyView() string {
	if m.response == nil || len(m.response.Body) == 0 {
		return dimStyle.Render("(empty body)")
	}

	raw := formatBody(m.response, m.bodyWidth())
	lines := strings.Split(raw, "\n")

	// Word wrap
	if m.wordWrap {
		lines = wrapLines(lines, m.bodyWidth())
	}

	// Build search match set
	matchSet := make(map[int]bool)
	for _, idx := range m.searchHits {
		matchSet[idx] = true
	}
	currentMatch := -1
	if len(m.searchHits) > 0 {
		currentMatch = m.searchHits[m.searchIdx]
	}

	lineNumWidth := len(fmt.Sprintf("%d", len(lines)))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#585858"))
	matchStyle := lipgloss.NewStyle().Background(lipgloss.Color("#3D3D00")).Foreground(lipgloss.Color("#FFFF00"))
	currentMatchStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FF9800")).Foreground(lipgloss.Color("#000000"))

	var sb strings.Builder
	for i, line := range lines {
		// Line number
		if m.showLineNums {
			num := fmt.Sprintf("%*d │ ", lineNumWidth, i+1)
			sb.WriteString(lineNumStyle.Render(num))
		}

		// Highlight search matches
		if m.searchQuery != "" && matchSet[i] {
			if i == currentMatch {
				line = highlightLine(line, m.searchQuery, currentMatchStyle)
			} else {
				line = highlightLine(line, m.searchQuery, matchStyle)
			}
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m DetailModel) bodyWidth() int {
	w := m.width - 4 // border padding
	if m.showLineNums {
		w -= 8 // line number column
	}
	if w < 20 {
		w = 20
	}
	return w
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

// --- Rendering helpers ---

func requestView(req *model.Request) string {
	var sb strings.Builder
	method := lipgloss.NewStyle().Foreground(methodColor(req.Method)).Bold(true).Render(req.Method)
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

func bodyInfo(resp *model.Response) string {
	size := len(resp.Body)
	var sizeStr string
	switch {
	case size == 0:
		return "0 B"
	case size < 1024:
		sizeStr = fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	ct := resp.ContentType
	if idx := strings.Index(ct, ";"); idx > 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	if ct == "" {
		return sizeStr
	}
	// Shorten common content types
	ct = strings.TrimPrefix(ct, "application/")
	ct = strings.TrimPrefix(ct, "text/")
	return fmt.Sprintf("%s  %s", ct, sizeStr)
}

func headersView(resp *model.Response) string {
	var sb strings.Builder
	for _, h := range resp.Headers {
		key := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB")).Render(h.Key)
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, h.Value))
	}
	return sb.String()
}

// formatBody renders the response body with syntax-appropriate formatting.
func formatBody(resp *model.Response, maxWidth int) string {
	if len(resp.Body) == 0 {
		return ""
	}
	ct := strings.ToLower(resp.ContentType)

	switch {
	case strings.Contains(ct, "json"):
		formatted := pretty.Pretty(resp.Body)
		return string(pretty.Color(formatted, nil))

	case strings.Contains(ct, "xml"), strings.Contains(ct, "html"):
		return indentXML(string(resp.Body))

	default:
		return string(resp.Body)
	}
}

// indentXML does a best-effort pretty-print of XML/HTML content.
func indentXML(s string) string {
	decoder := xml.NewDecoder(strings.NewReader(s))
	var sb strings.Builder
	depth := 0
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	attrKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB"))
	attrValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		indent := strings.Repeat("  ", depth)
		switch t := tok.(type) {
		case xml.StartElement:
			tag := tagStyle.Render("<" + t.Name.Local)
			for _, a := range t.Attr {
				tag += " " + attrKeyStyle.Render(a.Name.Local) + "=" + attrValStyle.Render(`"`+a.Value+`"`)
			}
			tag += tagStyle.Render(">")
			sb.WriteString(indent + tag + "\n")
			depth++
		case xml.EndElement:
			depth--
			if depth < 0 {
				depth = 0
			}
			indent = strings.Repeat("  ", depth)
			sb.WriteString(indent + tagStyle.Render("</"+t.Name.Local+">") + "\n")
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				sb.WriteString(indent + "  " + text + "\n")
			}
		}
	}

	result := sb.String()
	if result == "" {
		// XML parsing failed — return raw
		return s
	}
	return result
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
		clr  string
	}{
		{"DNS    ", t.DNS.Milliseconds(), "#89B4FA"},
		{"Connect", t.Connect.Milliseconds(), "#A6E3A1"},
		{"TLS    ", t.TLS.Milliseconds(), "#F9E2AF"},
		{"TTFB   ", t.TTFB.Milliseconds(), "#FAB387"},
		{"Body   ", t.BodyRead.Milliseconds(), "#CBA6F7"},
		{"Total  ", total, "#CDD6F4"},
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
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(p.clr)).Render(strings.Repeat("█", filled))
		bar += dimStyle.Render(strings.Repeat("░", empty))
		msStr := fmt.Sprintf("%dms", p.ms)
		sb.WriteString(fmt.Sprintf("%s  %s  %s\n", dimStyle.Render(p.name), bar, msStr))
	}
	return sb.String()
}

// --- Utility functions ---

// wrapLines wraps lines that exceed maxWidth, preserving line indices for search.
func wrapLines(lines []string, maxWidth int) []string {
	if maxWidth <= 0 {
		return lines
	}
	var wrapped []string
	for _, line := range lines {
		// Strip ANSI for length measurement but keep styled text
		plain := stripANSI(line)
		if len(plain) <= maxWidth {
			wrapped = append(wrapped, line)
			continue
		}
		// Naive wrap at maxWidth characters for plain text
		for len(plain) > maxWidth {
			wrapped = append(wrapped, plain[:maxWidth])
			plain = plain[maxWidth:]
		}
		if len(plain) > 0 {
			wrapped = append(wrapped, plain)
		}
	}
	return wrapped
}

// stripANSI removes ANSI escape sequences for accurate length measurement.
func stripANSI(s string) string {
	var sb strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if r == '\033' {
			inEsc = true
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// highlightLine highlights occurrences of query in line using the given style.
func highlightLine(line, query string, style lipgloss.Style) string {
	if query == "" {
		return line
	}
	lower := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	var sb strings.Builder
	pos := 0
	for {
		idx := strings.Index(lower[pos:], lowerQuery)
		if idx < 0 {
			sb.WriteString(line[pos:])
			break
		}
		sb.WriteString(line[pos : pos+idx])
		matchEnd := pos + idx + len(query)
		sb.WriteString(style.Render(line[pos+idx : matchEnd]))
		pos = matchEnd
	}
	return sb.String()
}
