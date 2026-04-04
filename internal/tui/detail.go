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
	tabBody    detailTab = iota // 1 — body first
	tabHeaders                  // 2
	tabTiming                   // 3
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

	// Body viewer
	wordWrap     bool
	showLineNums bool
	prettyPrint  bool   // true = formatted, false = raw
	searching    bool
	searchQuery  string
	searchHits   []int
	searchIdx    int
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
		prettyPrint:  true,
		tab:          tabBody, // body is default
	}
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) viewableHeight() int {
	h := m.height - 5 // sticky status + tab bar + padding
	if m.searching {
		h--
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
		m.tab = tabBody // always show body after response
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
		m.tab = tabBody
		m.offset = 0
	case "2":
		m.tab = tabHeaders
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

	// Scrolling
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
		m.offset = 999999

	// Body controls
	case "w":
		m.wordWrap = !m.wordWrap
		m.offset = 0
	case "l":
		m.showLineNums = !m.showLineNums
	case "p":
		m.prettyPrint = !m.prettyPrint
		m.offset = 0
	case "f":
		if m.response != nil && m.tab == tabBody {
			m.searching = true
			m.searchQuery = ""
			m.searchHits = nil
			m.searchIdx = 0
		}
	case "n":
		if len(m.searchHits) > 0 {
			m.searchIdx = (m.searchIdx + 1) % len(m.searchHits)
			m.offset = m.searchHits[m.searchIdx]
		}
	case "N":
		if len(m.searchHits) > 0 {
			m.searchIdx = (m.searchIdx - 1 + len(m.searchHits)) % len(m.searchHits)
			m.offset = m.searchHits[m.searchIdx]
		}
	}
	return m, nil
}

// --- Search ---

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

	// ── Sticky status bar ──
	sb.WriteString(m.stickyStatus())
	sb.WriteString("\n")

	// ── Tab bar ──
	sb.WriteString(m.tabBar())
	sb.WriteString("\n\n")

	// ── Tab content ──
	var content string
	switch m.tab {
	case tabBody:
		content = m.enhancedBodyView()
	case tabHeaders:
		content = headersView(m.response)
	case tabTiming:
		content = timingView(m.response)
	}

	lines := strings.Split(content, "\n")

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

	// Scroll position
	if len(lines) > m.viewableHeight() {
		pct := 0
		if maxOffset > 0 {
			pct = m.offset * 100 / maxOffset
		}
		sb.WriteString(dimStyle.Render(fmt.Sprintf("── %d%% (%d/%d) ──", pct, m.offset+1, len(lines))))
	}

	// Search bar
	if m.searching {
		sb.WriteString("\n")
		matchInfo := ""
		if m.searchQuery != "" {
			if len(m.searchHits) == 0 {
				matchInfo = " [no match]"
			} else {
				matchInfo = fmt.Sprintf(" [%d/%d]", m.searchIdx+1, len(m.searchHits))
			}
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(colorBorderActive).Render("find: " + m.searchQuery + "█" + matchInfo))
	}

	return sb.String()
}

// stickyStatus renders the always-visible status bar at the top of the response view.
func (m DetailModel) stickyStatus() string {
	resp := m.response
	if resp == nil {
		return ""
	}

	// Status code with color
	code := resp.StatusCode
	var clr color.Color
	var icon string
	switch {
	case code >= 200 && code < 300:
		clr = lipgloss.Color("#4CAF50")
		icon = "✓"
	case code >= 300 && code < 400:
		clr = lipgloss.Color("#FFFF00")
		icon = "→"
	case code >= 400 && code < 500:
		clr = lipgloss.Color("#FF9800")
		icon = "✗"
	default:
		clr = lipgloss.Color("#F44336")
		icon = "✗"
	}

	statusStyle := lipgloss.NewStyle().Foreground(clr).Bold(true)
	status := statusStyle.Render(fmt.Sprintf("%s %d %s", icon, code, resp.Status))

	// Content type (shortened)
	ct := resp.ContentType
	if idx := strings.Index(ct, ";"); idx > 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = strings.TrimPrefix(ct, "application/")
	ct = strings.TrimPrefix(ct, "text/")

	// Body size
	size := len(resp.Body)
	var sizeStr string
	switch {
	case size == 0:
		sizeStr = "0 B"
	case size < 1024:
		sizeStr = fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}

	// Timing (total only)
	timing := ""
	if resp.Timing.Total > 0 {
		ms := resp.Timing.Total.Milliseconds()
		if ms < 1000 {
			timing = fmt.Sprintf("%dms", ms)
		} else {
			timing = fmt.Sprintf("%.1fs", float64(ms)/1000)
		}
	}

	// Compose: ✓ 200 OK  ──  json  2.4 KB  142ms
	sep := dimStyle.Render(" ── ")
	parts := []string{status}
	if ct != "" {
		parts = append(parts, dimStyle.Render(ct))
	}
	parts = append(parts, dimStyle.Render(sizeStr))
	if timing != "" {
		parts = append(parts, dimStyle.Render(timing))
	}

	return strings.Join(parts, sep)
}

func (m DetailModel) tabBar() string {
	type tabDef struct {
		key   string
		label string
		tab   detailTab
	}
	tabs := []tabDef{
		{"1", "Body", tabBody},
		{"2", "Headers", tabHeaders},
		{"3", "Timing", tabTiming},
	}

	var sb strings.Builder
	for i, t := range tabs {
		label := t.key + ":" + t.label
		if t.tab == m.tab {
			sb.WriteString(lipgloss.NewStyle().
				Underline(true).
				Bold(true).
				Foreground(lipgloss.Color("#CDD6F4")).
				Render(label))
		} else {
			sb.WriteString(dimStyle.Render(label))
		}
		if i < len(tabs)-1 {
			sb.WriteString("  ")
		}
	}

	// Body tab indicators
	if m.tab == tabBody && m.response != nil {
		var extras []string
		if !m.prettyPrint {
			extras = append(extras, "raw")
		}
		if m.wordWrap {
			extras = append(extras, "wrap")
		}
		if m.searchQuery != "" && !m.searching {
			extras = append(extras, "/"+m.searchQuery)
		}
		if len(extras) > 0 {
			sb.WriteString("  " + dimStyle.Render("["+strings.Join(extras, " ")+"]"))
		}
	}
	return sb.String()
}

func (m DetailModel) getBodyText() string {
	if m.response == nil || len(m.response.Body) == 0 {
		return ""
	}
	if m.prettyPrint {
		return formatBody(m.response, m.bodyWidth())
	}
	return string(m.response.Body)
}

func (m DetailModel) enhancedBodyView() string {
	if m.response == nil || len(m.response.Body) == 0 {
		return dimStyle.Render("(empty body)")
	}

	var raw string
	if m.prettyPrint {
		raw = formatBody(m.response, m.bodyWidth())
	} else {
		raw = string(m.response.Body)
	}
	lines := strings.Split(raw, "\n")

	if m.wordWrap {
		lines = wrapLines(lines, m.bodyWidth())
	}

	// Search match set
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
		if m.showLineNums {
			num := fmt.Sprintf("%*d │ ", lineNumWidth, i+1)
			sb.WriteString(lineNumStyle.Render(num))
		}

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
	w := m.width - 4
	if m.showLineNums {
		w -= 8
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

func headersView(resp *model.Response) string {
	var sb strings.Builder
	for _, h := range resp.Headers {
		key := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB")).Render(h.Key)
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, h.Value))
	}
	return sb.String()
}

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
	if sb.Len() == 0 {
		return s
	}
	return sb.String()
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
		sb.WriteString(fmt.Sprintf("%s  %s  %dms\n", dimStyle.Render(p.name), bar, p.ms))
	}
	return sb.String()
}

// --- Utility ---

func wrapLines(lines []string, maxWidth int) []string {
	if maxWidth <= 0 {
		return lines
	}
	var wrapped []string
	for _, line := range lines {
		plain := stripANSI(line)
		if len(plain) <= maxWidth {
			wrapped = append(wrapped, line)
			continue
		}
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
