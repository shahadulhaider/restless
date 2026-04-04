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

	"github.com/shahadulhaider/restless/internal/assert"
	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/exporter"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/shahadulhaider/restless/internal/writer"
)

// section identifies a collapsible section (used for both request and response).
type section int

const (
	sectionBody    section = iota
	sectionHeaders
	sectionTiming // response only; request uses sectionMeta in slot 3
)

// detailMode tracks whether we're viewing request or response.
type detailMode int

const (
	modeRequest  detailMode = iota
	modeResponse
)

type DetailModel struct {
	request  *model.Request
	response *model.Response
	mode     detailMode // current view: request or response
	sending  bool
	errMsg   string
	width    int
	height   int
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

	// Request accordion state
	reqFolds   [4]bool // Body, Headers, Metadata, Assertions expanded
	reqOffset  int

	// Response accordion state
	respFolds  [4]bool // Body, Headers, Timing, Assertions expanded
	respOffset int

	expandAll bool
	pendingZ  bool
	pendingY  bool
	pendingYG bool // waiting for language key after yg

	// Body viewer
	wordWrap     bool
	showLineNums bool
	prettyPrint  bool
	searching    bool
	searchQuery  string
	searchHits   []int
	searchIdx    int
}

type responseReceived struct {
	resp *model.Response
	err  error
}

type yankResult struct {
	label string
	err   error
}

type historyLoadedMsg struct {
	entries []history.HistoryEntry
}

// Package-level section ranges (View is value receiver)
var lastSectionLines []sectionRange

func NewDetailModel(rootDir string, chainCtx *parser.ChainContext, cookies *engine.CookieManager) DetailModel {
	return DetailModel{
		rootDir:      rootDir,
		chainCtx:     chainCtx,
		cookies:      cookies,
		envVars:      make(map[string]string),
		showLineNums: true,
		prettyPrint:  true,
		mode:         modeRequest,
		reqFolds:     [4]bool{true, false, false, false},  // body expanded
		respFolds:    [4]bool{true, false, false, false},   // body expanded
	}
}

func (m DetailModel) Init() tea.Cmd { return nil }

func (m DetailModel) viewableHeight() int {
	h := m.height - 4 // toggle bar + sticky header + padding
	if m.searching {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (m *DetailModel) offset() int {
	if m.mode == modeRequest {
		return m.reqOffset
	}
	return m.respOffset
}

func (m *DetailModel) setOffset(v int) {
	if m.mode == modeRequest {
		m.reqOffset = v
	} else {
		m.respOffset = v
	}
}

func (m *DetailModel) folds() *[4]bool {
	if m.mode == modeRequest {
		return &m.reqFolds
	}
	return &m.respFolds
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
		m.mode = modeRequest
		m.reqOffset = 0
		m.respOffset = 0
		m.errMsg = ""
		m.showHistory = false
		m.clearSearch()
		m.reqFolds = [4]bool{true, false, false, false}
		m.respFolds = [4]bool{true, false, false, false}
		m.expandAll = false

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
			m.mode = modeResponse
			// Run assertions
			if m.request != nil && len(m.request.Assertions) > 0 {
				m.response.AssertionResults = assert.EvaluateAll(m.request, m.response)
			}
		}
		m.respOffset = 0
		hasAssertions := m.response != nil && len(m.response.AssertionResults) > 0
		if hasAssertions && !assert.AllPassed(m.response.AssertionResults) {
			// Auto-expand assertions section if any failed
			m.respFolds = [4]bool{true, false, false, true}
		} else {
			m.respFolds = [4]bool{true, false, false, false}
		}
		m.expandAll = false
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
			m.mode = modeResponse
			m.respOffset = 0
			m.respFolds = [4]bool{true, false, false, false}
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
			m.setOffset(m.searchHits[m.searchIdx])
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

	// z-prefix
	if m.pendingZ {
		m.pendingZ = false
		cur := m.sectionAtOffset()
		folds := m.folds()
		switch key {
		case "o":
			m.expandSection(cur)
		case "c":
			folds[cur] = false
		case "M":
			*folds = [4]bool{false, false, false, false}
			m.expandAll = false
		case "R":
			*folds = [4]bool{true, true, true, true}
			m.expandAll = true
		}
		return m, nil
	}

	// yg-prefix (code generation)
	if m.pendingYG {
		m.pendingYG = false
		return m.handleCodeGen(key)
	}

	// y-prefix
	if m.pendingY {
		m.pendingY = false
		if key == "g" {
			m.pendingYG = true
			return m, nil
		}
		return m.handleYank(key)
	}

	switch key {
	case "z":
		m.pendingZ = true
		return m, nil
	case "y":
		m.pendingY = true
		return m, nil

	// Request/Response toggle
	case "r":
		m.mode = modeRequest
		return m, nil
	case "s":
		if m.response != nil {
			m.mode = modeResponse
		}
		return m, nil

	// Section toggles
	case "1":
		m.toggleSection(0)
		m.setOffset(0)
	case "2":
		m.toggleSection(1)
	case "3":
		m.toggleSection(2)
	case " ":
		cur := m.sectionAtOffset()
		m.toggleSection(int(cur))

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
				// Merge file-level inline variables with env variables
				mergedVars := make(map[string]string)
				for k, v := range envVars {
					mergedVars[k] = v
				}
				if req.SourceFile != "" {
					if fileVars, err := parser.ExtractFileVariablesFromFile(req.SourceFile); err == nil {
						for k, v := range fileVars {
							if _, exists := mergedVars[k]; !exists {
								mergedVars[k] = v // file vars don't override env vars
							}
						}
					}
				}
				resolved, _ := parser.ResolveRequest(req, mergedVars, chainCtx)
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
		m.setOffset(m.offset() + 1)
	case "k", "up":
		if m.offset() > 0 {
			m.setOffset(m.offset() - 1)
		}
	case "ctrl+d":
		m.setOffset(m.offset() + m.viewableHeight()/2)
	case "ctrl+u":
		off := m.offset() - m.viewableHeight()/2
		if off < 0 {
			off = 0
		}
		m.setOffset(off)
	case "g":
		m.setOffset(0)
	case "G":
		m.setOffset(999999)

	// Body controls
	case "w":
		m.wordWrap = !m.wordWrap
		m.setOffset(0)
	case "l":
		m.showLineNums = !m.showLineNums
	case "p":
		m.prettyPrint = !m.prettyPrint
		m.setOffset(0)
	case "f":
		m.searching = true
		m.searchQuery = ""
		m.searchHits = nil
		m.searchIdx = 0
	case "n":
		if len(m.searchHits) > 0 {
			m.searchIdx = (m.searchIdx + 1) % len(m.searchHits)
			m.setOffset(m.searchHits[m.searchIdx])
		}
	case "N":
		if len(m.searchHits) > 0 {
			m.searchIdx = (m.searchIdx - 1 + len(m.searchHits)) % len(m.searchHits)
			m.setOffset(m.searchHits[m.searchIdx])
		}
	}
	return m, nil
}

func (m *DetailModel) expandSection(idx section) {
	folds := m.folds()
	if !m.expandAll {
		*folds = [4]bool{false, false, false, false}
	}
	folds[idx] = true
}

func (m *DetailModel) toggleSection(idx int) {
	folds := m.folds()
	if folds[idx] {
		folds[idx] = false
	} else {
		m.expandSection(section(idx))
	}
}

func (m DetailModel) sectionAtOffset() section {
	off := m.offset()
	for _, sr := range lastSectionLines {
		if off >= sr.start && off <= sr.end {
			return sr.sec
		}
	}
	return sectionBody
}

// --- Yank ---

func (m DetailModel) handleYank(key string) (DetailModel, tea.Cmd) {
	var text, label string

	if m.mode == modeRequest {
		// Yank from request
		switch key {
		case "b":
			if m.request != nil {
				text = m.request.Body
				label = "request body"
			}
		case "h":
			if m.request != nil {
				var sb strings.Builder
				for _, h := range m.request.Headers {
					sb.WriteString(h.Key + ": " + h.Value + "\n")
				}
				text = sb.String()
				label = "request headers"
			}
		case "a":
			if m.request != nil {
				text = writer.SerializeRequest(*m.request)
				label = "request"
			}
		case "c":
			if m.request != nil {
				text = exporter.ToCurl(*m.request)
				label = "curl"
			}
		}
	} else {
		// Yank from response
		switch key {
		case "b":
			if m.response != nil {
				if m.prettyPrint {
					text = formatBodyPlain(m.response)
				} else {
					text = string(m.response.Body)
				}
				label = "body"
			}
		case "h":
			if m.response != nil {
				var sb strings.Builder
				for _, h := range m.response.Headers {
					sb.WriteString(h.Key + ": " + h.Value + "\n")
				}
				text = sb.String()
				label = "headers"
			}
		case "a":
			if m.response != nil {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("HTTP %d %s\n", m.response.StatusCode, m.response.Status))
				for _, h := range m.response.Headers {
					sb.WriteString(h.Key + ": " + h.Value + "\n")
				}
				sb.WriteString("\n")
				sb.WriteString(string(m.response.Body))
				text = sb.String()
				label = "response"
			}
		case "c":
			if m.request != nil {
				text = exporter.ToCurl(*m.request)
				label = "curl"
			}
		}
	}

	if text == "" {
		return m, nil
	}
	return m, func() tea.Msg {
		err := exporter.CopyToClipboard(text)
		return yankResult{label: label, err: err}
	}
}

// --- Code Generation ---

func (m DetailModel) handleCodeGen(key string) (DetailModel, tea.Cmd) {
	if m.request == nil {
		return m, nil
	}

	gen, ok := exporter.Generators[key]
	if !ok {
		return m, nil
	}

	code := gen.Generate(*m.request)
	label := gen.Name + " code"

	return m, func() tea.Msg {
		err := exporter.CopyToClipboard(code)
		return yankResult{label: label, err: err}
	}
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
	if m.searchQuery == "" {
		return
	}
	content := m.currentAccordionContent()
	lines := strings.Split(content, "\n")
	query := strings.ToLower(m.searchQuery)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.searchHits = append(m.searchHits, i)
		}
	}
}

func (m DetailModel) currentAccordionContent() string {
	if m.mode == modeRequest {
		return m.buildRequestAccordion().content
	}
	return m.buildResponseAccordion().content
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
	}

	// ── Toggle bar (only when response exists) ──
	if m.response != nil {
		sb.WriteString(m.toggleBar())
		sb.WriteString("\n")
	}

	// ── Sticky header ──
	if m.mode == modeResponse && m.response != nil {
		sb.WriteString(m.stickyStatus())
		sb.WriteString("\n\n")
	} else if m.request != nil {
		method := lipgloss.NewStyle().Foreground(methodColor(m.request.Method)).Bold(true).Render(m.request.Method)
		sb.WriteString(fmt.Sprintf("%s %s\n\n", method, m.request.URL))
	}

	// ── Accordion content ──
	var result accordionResult
	if m.mode == modeRequest {
		result = m.buildRequestAccordion()
	} else {
		result = m.buildResponseAccordion()
	}
	lastSectionLines = result.ranges

	lines := strings.Split(result.content, "\n")
	off := m.offset()

	// Clamp
	maxOff := len(lines) - m.viewableHeight()
	if maxOff < 0 {
		maxOff = 0
	}
	if off > maxOff {
		off = maxOff
	}
	if off < 0 {
		off = 0
	}
	m.setOffset(off)

	end := off + m.viewableHeight()
	if end > len(lines) {
		end = len(lines)
	}
	for _, l := range lines[off:end] {
		sb.WriteString(l + "\n")
	}

	// Scroll indicator
	if len(lines) > m.viewableHeight() {
		pct := 0
		if maxOff > 0 {
			pct = off * 100 / maxOff
		}
		sb.WriteString(dimStyle.Render(fmt.Sprintf("── %d%% (%d/%d) ──", pct, off+1, len(lines))))
	}

	// Overlays
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
	if m.pendingZ {
		sb.WriteString("\n" + dimStyle.Render("z-"))
	}
	if m.pendingY {
		sb.WriteString("\n" + dimStyle.Render("y- (b:body  h:headers  a:all  c:curl  g:generate code)"))
	}
	if m.pendingYG {
		sb.WriteString("\n" + dimStyle.Render("yg- (p:python  j:javascript  g:go  v:java  r:ruby  h:httpie  c:curl  w:powershell)"))
	}

	return sb.String()
}

func (m DetailModel) toggleBar() string {
	reqStyle := dimStyle
	respStyle := dimStyle
	if m.mode == modeRequest {
		reqStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4")).Underline(true)
	} else {
		respStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4")).Underline(true)
	}
	return reqStyle.Render("[r] Request") + dimStyle.Render("  │  ") + respStyle.Render("[s] Response")
}

// --- Request accordion ---

func (m DetailModel) buildRequestAccordion() accordionResult {
	req := m.request
	if req == nil {
		return accordionResult{}
	}

	// Section 1: Body
	bodyContent := ""
	bodySummary := "(empty)"
	bodyPreview := ""
	if req.Body != "" {
		bodyContent = m.renderTextContent(req.Body)
		raw := strings.ReplaceAll(req.Body, "\n", " ")
		bodySummary = truncate(strings.Join(strings.Fields(raw), " "), 50)
		bodyPreview = previewLines(req.Body, 2, m.width)
	}

	// Section 2: Headers
	hdrContent := ""
	hdrSummary := "(none)"
	hdrPreview := ""
	if len(req.Headers) > 0 {
		hdrSummary = fmt.Sprintf("(%d)", len(req.Headers))
		var sb strings.Builder
		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB"))
		for _, h := range req.Headers {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render(h.Key), h.Value))
		}
		hdrContent = sb.String()
		if len(req.Headers) > 0 {
			hdrSummary += " ── " + req.Headers[0].Key + ": " + req.Headers[0].Value
		}
		limit := 2
		if limit > len(req.Headers) {
			limit = len(req.Headers)
		}
		var psb strings.Builder
		for _, h := range req.Headers[:limit] {
			psb.WriteString("  " + dimStyle.Render(h.Key+": "+h.Value) + "\n")
		}
		hdrPreview = strings.TrimRight(psb.String(), "\n")
	}

	// Section 3: Metadata
	metaContent := ""
	metaSummary := ""
	var metaParts []string
	if req.Name != "" {
		metaParts = append(metaParts, "@name "+req.Name)
	}
	if req.Metadata.NoRedirect {
		metaParts = append(metaParts, "@no-redirect")
	}
	if req.Metadata.NoCookieJar {
		metaParts = append(metaParts, "@no-cookie-jar")
	}
	if req.Metadata.Timeout > 0 {
		metaParts = append(metaParts, fmt.Sprintf("@timeout %ds", int(req.Metadata.Timeout.Seconds())))
	}
	if req.Metadata.ConnTimeout > 0 {
		metaParts = append(metaParts, fmt.Sprintf("@connection-timeout %ds", int(req.Metadata.ConnTimeout.Seconds())))
	}
	if len(metaParts) > 0 {
		metaSummary = strings.Join(metaParts, ", ")
		var sb strings.Builder
		for _, p := range metaParts {
			sb.WriteString("  " + dimStyle.Render("# "+p) + "\n")
		}
		metaContent = sb.String()
	} else {
		metaSummary = "(none)"
	}

	sections := []accordionSection{
		{key: "1", label: "Body", summary: bodySummary, preview: bodyPreview, content: bodyContent, expanded: m.reqFolds[0]},
		{key: "2", label: fmt.Sprintf("Headers (%d)", len(req.Headers)), summary: hdrSummary, preview: hdrPreview, content: hdrContent, expanded: m.reqFolds[1]},
		{key: "3", label: "Metadata", summary: metaSummary, content: metaContent, expanded: m.reqFolds[2]},
	}

	return renderAccordionSections(sections, m.width)
}

// --- Response accordion ---

func (m DetailModel) buildResponseAccordion() accordionResult {
	resp := m.response
	if resp == nil {
		return accordionResult{}
	}

	// Section 1: Body
	bodyContent := ""
	bodySummary := "(empty)"
	bodyPreview := ""
	if len(resp.Body) > 0 {
		bodyContent = m.renderResponseBodyContent()
		raw := string(resp.Body)
		raw = strings.ReplaceAll(raw, "\n", " ")
		bodySummary = truncate(strings.Join(strings.Fields(raw), " "), 50)
		var src string
		if m.prettyPrint {
			src = formatBody(resp, m.bodyWidth())
		} else {
			src = string(resp.Body)
		}
		bodyPreview = previewLines(src, 2, m.width)
	}

	// Section 2: Headers
	hdrContent := ""
	hdrSummary := fmt.Sprintf("(%d)", len(resp.Headers))
	hdrPreview := ""
	if len(resp.Headers) > 0 {
		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB"))
		var sb strings.Builder
		for _, h := range resp.Headers {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render(h.Key), h.Value))
		}
		hdrContent = sb.String()
		hdrSummary += " ── " + resp.Headers[0].Key + ": " + resp.Headers[0].Value
		limit := 2
		if limit > len(resp.Headers) {
			limit = len(resp.Headers)
		}
		var psb strings.Builder
		for _, h := range resp.Headers[:limit] {
			psb.WriteString("  " + dimStyle.Render(h.Key+": "+h.Value) + "\n")
		}
		hdrPreview = strings.TrimRight(psb.String(), "\n")
	}

	// Section 3: Timing
	totalMs := resp.Timing.Total.Milliseconds()
	timingContent := timingView(resp)
	timingSummary := m.timingPreviewStr()

	sections := []accordionSection{
		{key: "1", label: "Body", summary: bodySummary, preview: bodyPreview, content: bodyContent, expanded: m.respFolds[0]},
		{key: "2", label: fmt.Sprintf("Headers (%d)", len(resp.Headers)), summary: hdrSummary, preview: hdrPreview, content: hdrContent, expanded: m.respFolds[1]},
		{key: "3", label: fmt.Sprintf("Timing ── %dms", totalMs), summary: timingSummary, content: timingContent, expanded: m.respFolds[2]},
	}

	// Section 4: Assertions (only if assertions exist)
	if len(resp.AssertionResults) > 0 {
		passed := assert.CountPassed(resp.AssertionResults)
		total := len(resp.AssertionResults)
		allOk := assert.AllPassed(resp.AssertionResults)

		assertLabel := fmt.Sprintf("Assertions (%d/%d passed)", passed, total)
		assertSummary := ""
		if !allOk {
			// Show first failing assertion in summary
			for _, r := range resp.AssertionResults {
				if !r.Passed {
					assertSummary = "✗ " + r.Assertion.Raw
					break
				}
			}
		} else {
			assertSummary = "all passed"
		}

		var assertContent strings.Builder
		passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F44336"))
		for _, r := range resp.AssertionResults {
			if r.Passed {
				assertContent.WriteString("  " + passStyle.Render("✓") + " " + r.Assertion.Raw + "\n")
			} else {
				line := "  " + failStyle.Render("✗") + " " + r.Assertion.Raw
				if r.Error != "" {
					line += dimStyle.Render(fmt.Sprintf(" (%s)", r.Error))
				} else {
					line += dimStyle.Render(fmt.Sprintf(" (got %s)", r.Actual))
				}
				assertContent.WriteString(line + "\n")
			}
		}

		assertPreview := ""
		if !allOk {
			// Preview: first failing assertion
			for _, r := range resp.AssertionResults {
				if !r.Passed {
					assertPreview = "  " + dimStyle.Render("✗ "+r.Assertion.Raw+" (got "+r.Actual+")")
					break
				}
			}
		}

		sections = append(sections, accordionSection{
			key: "4", label: assertLabel, summary: assertSummary,
			preview: assertPreview, content: assertContent.String(),
			expanded: m.respFolds[3],
		})
	}

	return renderAccordionSections(sections, m.width)
}

// --- Body rendering ---

func (m DetailModel) renderTextContent(body string) string {
	lines := strings.Split(body, "\n")
	if m.wordWrap {
		lines = wrapLines(lines, m.bodyWidth())
	}

	lineNumWidth := len(fmt.Sprintf("%d", len(lines)))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#585858"))

	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString("  ")
		if m.showLineNums {
			sb.WriteString(lineNumStyle.Render(fmt.Sprintf("%*d │ ", lineNumWidth, i+1)))
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func (m DetailModel) renderResponseBodyContent() string {
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
			sb.WriteString(lineNumStyle.Render(fmt.Sprintf("%*d │ ", lineNumWidth, i+1)))
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

func (m DetailModel) bodyWidth() int {
	w := m.width - 6
	if m.showLineNums {
		w -= 8
	}
	if w < 20 {
		w = 20
	}
	return w
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
	status := lipgloss.NewStyle().Foreground(clr).Bold(true).Render(fmt.Sprintf("%s %d %s", icon, code, resp.Status))

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

func (m DetailModel) timingPreviewStr() string {
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
	return strings.Join(parts, " → ")
}

// --- History ---

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

// --- Format helpers ---

func formatBodyPlain(resp *model.Response) string {
	if len(resp.Body) == 0 {
		return ""
	}
	ct := strings.ToLower(resp.ContentType)
	if strings.Contains(ct, "json") {
		return string(pretty.Pretty(resp.Body))
	}
	return string(resp.Body)
}

func formatBody(resp *model.Response, maxWidth int) string {
	if len(resp.Body) == 0 {
		return ""
	}
	ct := strings.ToLower(resp.ContentType)
	switch {
	case strings.Contains(ct, "json"):
		return string(pretty.Color(pretty.Pretty(resp.Body), nil))
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
			sb.WriteString(strings.Repeat("  ", depth) + tagStyle.Render("</"+t.Name.Local+">") + "\n")
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
