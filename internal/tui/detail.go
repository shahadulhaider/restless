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

// section identifies a collapsible response section.
type section int

const (
	sectionBody    section = iota
	sectionHeaders
	sectionTiming
)

type DetailModel struct {
	request  *model.Request
	response *model.Response
	sending  bool
	errMsg   string
	width    int
	height   int
	offset   int
	rootDir  string

	currentEnv string
	envVars    map[string]string
	chainCtx   *parser.ChainContext
	cookies    *engine.CookieManager

	showHistory    bool
	historyEntries []history.HistoryEntry
	historyIdx     int
	diffMode       bool
	diffIdxA       int

	// Section fold state
	bodyExpanded    bool
	headersExpanded bool
	timingExpanded  bool
	expandAll       bool // zR override — allows multiple sections open
	pendingZ        bool // waiting for second char in z-command

	// Body viewer
	wordWrap     bool
	showLineNums bool
	prettyPrint  bool
	searching    bool
	searchQuery  string
	searchHits   []int
	searchIdx    int

	// Rendered section line ranges (computed each View)
	sectionLines []sectionRange
}

type sectionRange struct {
	sec   section
	start int // line index in rendered output
	end   int
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
		rootDir:         rootDir,
		chainCtx:        chainCtx,
		cookies:         cookies,
		envVars:         make(map[string]string),
		showLineNums:    true,
		prettyPrint:     true,
		bodyExpanded:    true,
		headersExpanded: false,
		timingExpanded:  false,
	}
}

func (m DetailModel) Init() tea.Cmd { return nil }

func (m DetailModel) viewableHeight() int {
	h := m.height - 3 // sticky status + padding
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
		m.resetFolds()

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
		m.resetFolds()

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

func (m *DetailModel) resetFolds() {
	m.bodyExpanded = true
	m.headersExpanded = false
	m.timingExpanded = false
	m.expandAll = false
}

// expandSection expands sec. In accordion mode, collapses others.
func (m *DetailModel) expandSection(sec section) {
	if !m.expandAll {
		m.bodyExpanded = false
		m.headersExpanded = false
		m.timingExpanded = false
	}
	switch sec {
	case sectionBody:
		m.bodyExpanded = true
	case sectionHeaders:
		m.headersExpanded = true
	case sectionTiming:
		m.timingExpanded = true
	}
}

func (m *DetailModel) collapseSection(sec section) {
	switch sec {
	case sectionBody:
		m.bodyExpanded = false
	case sectionHeaders:
		m.headersExpanded = false
	case sectionTiming:
		m.timingExpanded = false
	}
}

func (m *DetailModel) toggleSection(sec section) {
	expanded := false
	switch sec {
	case sectionBody:
		expanded = m.bodyExpanded
	case sectionHeaders:
		expanded = m.headersExpanded
	case sectionTiming:
		expanded = m.timingExpanded
	}
	if expanded {
		m.collapseSection(sec)
	} else {
		m.expandSection(sec)
	}
}

// sectionAtOffset returns which section the current scroll offset is in.
func (m DetailModel) sectionAtOffset() section {
	for _, sr := range lastSectionLines {
		if m.offset >= sr.start && m.offset <= sr.end {
			return sr.sec
		}
	}
	return sectionBody
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
			m.offset = 0
			m.resetFolds()
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
	key := msg.String()

	// Handle z-prefix commands
	if m.pendingZ {
		m.pendingZ = false
		cur := m.sectionAtOffset()
		switch key {
		case "o": // zo — expand current section
			m.expandSection(cur)
		case "c": // zc — collapse current section
			m.collapseSection(cur)
		case "M": // zM — collapse all
			m.bodyExpanded = false
			m.headersExpanded = false
			m.timingExpanded = false
			m.expandAll = false
		case "R": // zR — expand all
			m.bodyExpanded = true
			m.headersExpanded = true
			m.timingExpanded = true
			m.expandAll = true
		}
		return m, nil
	}

	switch key {
	case "z":
		m.pendingZ = true
		return m, nil

	// Direct section toggles
	case "1":
		m.toggleSection(sectionBody)
		m.offset = 0
	case "2":
		m.toggleSection(sectionHeaders)
	case "3":
		m.toggleSection(sectionTiming)

	// Space toggles section under cursor
	case " ":
		if m.response != nil {
			cur := m.sectionAtOffset()
			m.toggleSection(cur)
		}

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
		if m.response != nil {
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
	// Search across the full rendered accordion
	rendered := m.renderAccordion()
	lines := strings.Split(rendered, "\n")
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

	// ── Sticky status ──
	sb.WriteString(m.stickyStatus())
	sb.WriteString("\n\n")

	// ── Accordion sections ──
	content := m.renderAccordion()
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

	// Scroll indicator
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

	// z-pending indicator
	if m.pendingZ {
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("z-"))
	}

	return sb.String()
}

// renderAccordion renders all three sections as a single scrollable view.
// It also populates m.sectionLines for section-at-offset detection.
func (m DetailModel) renderAccordion() string {
	resp := m.response
	if resp == nil {
		return ""
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4"))
	divider := dimStyle.Render(strings.Repeat("─", max(m.width-6, 10)))

	var sb strings.Builder
	var ranges []sectionRange
	lineIdx := 0

	countLines := func(s string) int {
		if s == "" {
			return 0
		}
		return strings.Count(s, "\n") + 1
	}

	// ── Body Section ──
	bodyStart := lineIdx
	if m.bodyExpanded {
		sb.WriteString(headerStyle.Render("▼ Body") + " " + divider + "\n")
		lineIdx++
		bodyContent := m.renderBodyContent()
		sb.WriteString(bodyContent)
		lineIdx += countLines(bodyContent)
		sb.WriteString("\n")
		lineIdx++
	} else {
		preview := m.bodyPreview()
		sb.WriteString(headerStyle.Render("▶ Body") + " " + dimStyle.Render("── "+preview) + "\n")
		lineIdx++
		// Show 2-line preview
		previewLines := m.bodyPreviewLines(2)
		if previewLines != "" {
			sb.WriteString(dimStyle.Render(previewLines) + "\n")
			lineIdx += countLines(previewLines)
		}
		sb.WriteString("\n")
		lineIdx++
	}
	ranges = append(ranges, sectionRange{sectionBody, bodyStart, lineIdx - 1})

	// ── Headers Section ──
	headersStart := lineIdx
	headerCount := len(resp.Headers)
	if m.headersExpanded {
		sb.WriteString(headerStyle.Render(fmt.Sprintf("▼ Headers (%d)", headerCount)) + " " + divider + "\n")
		lineIdx++
		hdrContent := headersView(resp)
		sb.WriteString(hdrContent)
		lineIdx += countLines(hdrContent)
		sb.WriteString("\n")
		lineIdx++
	} else {
		preview := m.headersPreview()
		sb.WriteString(headerStyle.Render(fmt.Sprintf("▶ Headers (%d)", headerCount)) + " " + dimStyle.Render("── "+preview) + "\n")
		lineIdx++
		previewLines := m.headersPreviewLines(2)
		if previewLines != "" {
			sb.WriteString(dimStyle.Render(previewLines) + "\n")
			lineIdx += countLines(previewLines)
		}
		sb.WriteString("\n")
		lineIdx++
	}
	ranges = append(ranges, sectionRange{sectionHeaders, headersStart, lineIdx - 1})

	// ── Timing Section ──
	timingStart := lineIdx
	totalMs := resp.Timing.Total.Milliseconds()
	if m.timingExpanded {
		sb.WriteString(headerStyle.Render(fmt.Sprintf("▼ Timing ── %dms", totalMs)) + " " + divider + "\n")
		lineIdx++
		timContent := timingView(resp)
		sb.WriteString(timContent)
		lineIdx += countLines(timContent)
	} else {
		preview := m.timingPreview()
		sb.WriteString(headerStyle.Render(fmt.Sprintf("▶ Timing ── %dms", totalMs)) + " " + dimStyle.Render("── "+preview) + "\n")
		lineIdx++
	}
	ranges = append(ranges, sectionRange{sectionTiming, timingStart, lineIdx})

	// Store ranges for sectionAtOffset (we can't modify m directly in View,
	// but this is used by the next Update call)
	// We'll use a package-level workaround since View is a value receiver
	lastSectionLines = ranges

	return sb.String()
}

// Package-level storage for section ranges (View is value receiver, can't modify m)
var lastSectionLines []sectionRange

func (m DetailModel) renderBodyContent() string {
	if len(m.response.Body) == 0 {
		return dimStyle.Render("  (empty body)")
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

	// Search matches
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
		sb.WriteString("  ")
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
		sb.WriteString(line + "\n")
	}

	// Mode indicators
	var indicators []string
	if !m.prettyPrint {
		indicators = append(indicators, "raw")
	}
	if m.wordWrap {
		indicators = append(indicators, "wrap")
	}
	if m.searchQuery != "" && !m.searching {
		indicators = append(indicators, "/"+m.searchQuery)
	}
	if len(indicators) > 0 {
		sb.WriteString("  " + dimStyle.Render("["+strings.Join(indicators, " ")+"]") + "\n")
	}

	return sb.String()
}

// --- Collapsed previews ---

func (m DetailModel) bodyPreview() string {
	if m.response == nil || len(m.response.Body) == 0 {
		return "(empty)"
	}
	raw := string(m.response.Body)
	raw = strings.ReplaceAll(raw, "\n", " ")
	raw = strings.Join(strings.Fields(raw), " ") // normalize whitespace
	if len(raw) > 50 {
		raw = raw[:50] + "..."
	}
	return raw
}

func (m DetailModel) bodyPreviewLines(n int) string {
	if m.response == nil || len(m.response.Body) == 0 {
		return ""
	}
	var raw string
	if m.prettyPrint {
		raw = formatBody(m.response, m.bodyWidth())
	} else {
		raw = string(m.response.Body)
	}
	lines := strings.Split(raw, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	var sb strings.Builder
	for _, l := range lines {
		if len(l) > m.width-8 {
			l = l[:m.width-8] + "..."
		}
		sb.WriteString("  " + l + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (m DetailModel) headersPreview() string {
	if m.response == nil || len(m.response.Headers) == 0 {
		return "(none)"
	}
	h := m.response.Headers[0]
	return h.Key + ": " + h.Value
}

func (m DetailModel) headersPreviewLines(n int) string {
	if m.response == nil || len(m.response.Headers) == 0 {
		return ""
	}
	var sb strings.Builder
	limit := n
	if limit > len(m.response.Headers) {
		limit = len(m.response.Headers)
	}
	for _, h := range m.response.Headers[:limit] {
		sb.WriteString("  " + dimStyle.Render(h.Key+": "+h.Value) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (m DetailModel) timingPreview() string {
	if m.response == nil {
		return ""
	}
	t := m.response.Timing
	var parts []string
	if t.DNS > 0 {
		parts = append(parts, fmt.Sprintf("DNS %dms", t.DNS.Milliseconds()))
	}
	if t.TLS > 0 {
		parts = append(parts, fmt.Sprintf("TLS %dms", t.TLS.Milliseconds()))
	}
	if t.TTFB > 0 {
		parts = append(parts, fmt.Sprintf("TTFB %dms", t.TTFB.Milliseconds()))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " → ")
}

// --- Sticky status ---

func (m DetailModel) stickyStatus() string {
	resp := m.response
	if resp == nil {
		return ""
	}

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

	ct := resp.ContentType
	if idx := strings.Index(ct, ";"); idx > 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = strings.TrimPrefix(ct, "application/")
	ct = strings.TrimPrefix(ct, "text/")

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

	timing := ""
	if resp.Timing.Total > 0 {
		ms := resp.Timing.Total.Milliseconds()
		if ms < 1000 {
			timing = fmt.Sprintf("%dms", ms)
		} else {
			timing = fmt.Sprintf("%.1fs", float64(ms)/1000)
		}
	}

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

// --- History view ---

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

func (m DetailModel) bodyWidth() int {
	w := m.width - 6 // indent + border
	if m.showLineNums {
		w -= 8
	}
	if w < 20 {
		w = 20
	}
	return w
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
		sb.WriteString(fmt.Sprintf("  %s: %s\n", key, h.Value))
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
		return dimStyle.Render("  (no timing data)")
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
		sb.WriteString(fmt.Sprintf("  %s  %s  %dms\n", dimStyle.Render(p.name), bar, p.ms))
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
