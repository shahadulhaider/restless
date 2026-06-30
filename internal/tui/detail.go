package tui

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/tidwall/pretty"

	"github.com/tidwall/gjson"

	"github.com/shahadulhaider/restless/internal/assert"
	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/exporter"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/shahadulhaider/restless/internal/script"
	"github.com/shahadulhaider/restless/internal/writer"
)

// detailMode tracks whether we're viewing request or response.
type detailMode int

const (
	modeRequest detailMode = iota
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
	showDiff       bool
	diffText       string

	// Tab state (replaces accordion folds)
	reqTab  int // active tab index for request view (0=Body, 1=Headers, 2=Metadata)
	respTab int // active tab index for response view (0=Body, 1=Headers, 2=Timing, 3=Assertions)

	reqOffset  int
	respOffset int

	pendingZ bool
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

	// Persistent cursor
	cursorLine int // persistent cursor line in accordion content
	cursorCol  int // persistent cursor column (rune index)

	// Visual selection mode (character-level)
	selecting       bool
	selectAnchor    int      // anchor line
	selectAnchorCol int      // anchor column (rune index)
	selectCursor    int      // cursor line (decoupled from scroll offset)
	selectCol       int      // cursor column (rune index)
	selectLines     []string // plain-text snapshot captured on entry
	selectLineMode  bool     // true = visual line mode (V)

	// JSON path jump
	jumpingPath bool
	jumpQuery   string

	// Structural JSON folding: collapsed node openers keyed by JSON-line index.
	respCollapsed map[int]bool
	reqCollapsed  map[int]bool

	// g-prefix (for gp)
	pendingG bool
}

type responseReceived struct {
	resp *model.Response
	err  error
}

type inlineEditRequest struct {
	request *model.Request
	focus   string // "body" or "headers"
}

type yankResult struct {
	label string
	err   error
}

type historyLoadedMsg struct {
	entries []history.HistoryEntry
}

var lastVisibleRange [2]int

func NewDetailModel(rootDir string, chainCtx *parser.ChainContext, cookies *engine.CookieManager) DetailModel {
	return DetailModel{
		rootDir:       rootDir,
		chainCtx:      chainCtx,
		cookies:       cookies,
		envVars:       make(map[string]string),
		showLineNums:  true,
		prettyPrint:   true,
		mode:          modeRequest,
		respCollapsed: make(map[int]bool),
		reqCollapsed:  make(map[int]bool),
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

// ScrollBy moves the active view offset by delta lines (clamped at the top; the
// bottom is clamped during View once content height is known).
func (m *DetailModel) ScrollBy(delta int) {
	off := m.offset() + delta
	if off < 0 {
		off = 0
	}
	m.setOffset(off)
}

func (m *DetailModel) activeTab() int {
	if m.mode == modeRequest {
		return m.reqTab
	}
	return m.respTab
}

func (m *DetailModel) setActiveTab(idx int) {
	if m.mode == modeRequest {
		m.reqTab = idx
	} else {
		m.respTab = idx
	}
	m.cursorLine = 0
	m.cursorCol = 0
	m.setOffset(0)
	m.selecting = false
	m.selectLines = nil
	m.selectLineMode = false
	m.clearSearch()
}

func (m DetailModel) tabCount() int {
	if m.mode == modeRequest {
		return 3
	}
	if m.response != nil && len(m.response.AssertionResults) > 0 {
		return 4
	}
	return 3
}

// InputActive reports whether the detail pane is capturing keystrokes for an
// inline input, overlay, or multi-key prefix.
func (m DetailModel) InputActive() bool {
	return m.searching || m.jumpingPath || m.selecting ||
		m.showHistory || m.showDiff ||
		m.pendingG || m.pendingZ || m.pendingY || m.pendingYG
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
		m.cursorLine = 0
		m.cursorCol = 0
		m.errMsg = ""
		m.showHistory = false
		m.clearSearch()
		m.reqTab = 0
		m.respTab = 0
		m.reqCollapsed = make(map[int]bool)
		m.respCollapsed = make(map[int]bool)

	case historyLoadedMsg:
		m.historyEntries = msg.entries
		m.historyIdx = 0
		m.diffMode = false

	case tea.PasteMsg:
		if m.searching {
			m.searchQuery += sanitizeInline(msg.Content)
			m.rebuildSearchHits()
		} else if m.jumpingPath {
			m.jumpQuery += sanitizeInline(msg.Content)
		}
		return m, nil

	case responseReceived:
		m.sending = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.response = msg.resp
			m.errMsg = ""
			m.mode = modeResponse
		}
		m.respOffset = 0
		m.cursorLine = 0
		m.cursorCol = 0
		hasAssertions := m.response != nil && len(m.response.AssertionResults) > 0
		if hasAssertions && !assert.AllPassed(m.response.AssertionResults) {
			m.respTab = 3
		} else {
			m.respTab = 0
		}
		m.respCollapsed = make(map[int]bool)
		m.clearSearch()

	case tea.KeyPressMsg:
		if m.showDiff {
			switch msg.String() {
			case "esc", "q":
				m.showDiff = false
				m.showHistory = true
			}
			return m, nil
		}
		if m.showHistory {
			return m.updateHistory(msg)
		}
		if m.searching {
			return m.updateSearch(msg)
		}
		if m.selecting {
			return m.updateSelecting(msg)
		}
		if m.jumpingPath {
			return m.updateJumpPath(msg)
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
			m.respTab = 0
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
			m.diffText = history.Diff(a, b)
			m.diffMode = false
			m.showDiff = true
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

func (m DetailModel) updateSelecting(msg tea.KeyPressMsg) (DetailModel, tea.Cmd) {
	last := len(m.selectLines) - 1
	switch msg.String() {
	case "esc":
		m.selecting = false
		m.selectLines = nil
		m.selectLineMode = false
	case "j", "down":
		if m.selectCursor < last {
			m.selectCursor++
			m.clampSelectCol()
			m.scrollToCursor()
		}
	case "k", "up":
		if m.selectCursor > 0 {
			m.selectCursor--
			m.clampSelectCol()
			m.scrollToCursor()
		}
	case "h", "left":
		if m.selectCol > 0 {
			m.selectCol--
		}
	case "l", "right":
		if m.selectCol < m.selectLineLen(m.selectCursor) {
			m.selectCol++
		}
	case "0":
		m.selectCol = 0
	case "$":
		m.selectCol = m.selectLineLen(m.selectCursor)
	case "w":
		m.selectCol = nextWordCol([]rune(m.selectLineAt(m.selectCursor)), m.selectCol)
	case "b":
		m.selectCol = prevWordCol([]rune(m.selectLineAt(m.selectCursor)), m.selectCol)
	case "ctrl+d":
		m.selectCursor = min(m.selectCursor+m.viewableHeight()/2, last)
		m.clampSelectCol()
		m.scrollToCursor()
	case "ctrl+u":
		m.selectCursor = max(m.selectCursor-m.viewableHeight()/2, 0)
		m.clampSelectCol()
		m.scrollToCursor()
	case "G":
		m.selectCursor = max(last, 0)
		m.clampSelectCol()
		m.scrollToCursor()
	case "y":
		text := m.selectedTextRange()
		m.selecting = false
		m.selectLines = nil
		m.selectLineMode = false
		if text == "" {
			return m, nil
		}
		return m, copyCmd(text, "selection")
	}
	return m, nil
}

func (m DetailModel) selectLineAt(i int) string {
	if i < 0 || i >= len(m.selectLines) {
		return ""
	}
	return m.selectLines[i]
}

func (m DetailModel) selectLineLen(i int) int {
	return len([]rune(m.selectLineAt(i)))
}

func (m *DetailModel) clampSelectCol() {
	if n := m.selectLineLen(m.selectCursor); m.selectCol > n {
		m.selectCol = n
	}
}

func (m *DetailModel) scrollToCursor() {
	vh := m.viewableHeight()
	if m.selectCursor < m.offset() {
		m.setOffset(m.selectCursor)
	} else if m.selectCursor >= m.offset()+vh {
		m.setOffset(m.selectCursor - vh + 1)
	}
}

// selectionBounds returns the selection extent normalized so (loL,loC) precedes
// (hiL,hiC) in reading order.
func (m DetailModel) selectionBounds() (loL, loC, hiL, hiC int) {
	if m.selectAnchor < m.selectCursor ||
		(m.selectAnchor == m.selectCursor && m.selectAnchorCol <= m.selectCol) {
		return m.selectAnchor, m.selectAnchorCol, m.selectCursor, m.selectCol
	}
	return m.selectCursor, m.selectCol, m.selectAnchor, m.selectAnchorCol
}

// lineSelCols returns the highlight span [a,b) for a line, inclusive of the
// character under the cursor. ok is false when the line is outside the selection.
func (m DetailModel) lineSelCols(lineIdx, lineLen int) (a, b int, ok bool) {
	loL, loC, hiL, hiC := m.selectionBounds()
	if lineIdx < loL || lineIdx > hiL {
		return 0, 0, false
	}
	if m.selectLineMode {
		return 0, lineLen, true
	}
	a, b = 0, lineLen
	if lineIdx == loL {
		a = loC
	}
	if lineIdx == hiL {
		b = hiC + 1
	}
	if a < 0 {
		a = 0
	}
	if b > lineLen {
		b = lineLen
	}
	if a > b {
		a = b
	}
	return a, b, true
}

func (m DetailModel) selectedTextRange() string {
	loL, loC, hiL, hiC := m.selectionBounds()
	if loL < 0 || loL >= len(m.selectLines) {
		return ""
	}
	if hiL >= len(m.selectLines) {
		hiL = len(m.selectLines) - 1
	}
	if loL == hiL {
		return runeSlice(m.selectLines[loL], loC, hiC+1)
	}
	var sb strings.Builder
	sb.WriteString(runeSlice(m.selectLines[loL], loC, -1))
	for i := loL + 1; i < hiL; i++ {
		sb.WriteString("\n" + m.selectLines[i])
	}
	sb.WriteString("\n" + runeSlice(m.selectLines[hiL], 0, hiC+1))
	return sb.String()
}

func runeSlice(s string, a, b int) string {
	r := []rune(s)
	if a < 0 {
		a = 0
	}
	if a > len(r) {
		a = len(r)
	}
	if b < 0 || b > len(r) {
		b = len(r)
	}
	if a > b {
		return ""
	}
	return string(r[a:b])
}

func nextWordCol(runes []rune, col int) int {
	n := len(runes)
	for col < n && !unicode.IsSpace(runes[col]) {
		col++
	}
	for col < n && unicode.IsSpace(runes[col]) {
		col++
	}
	return col
}

func prevWordCol(runes []rune, col int) int {
	for col > 0 && unicode.IsSpace(runes[col-1]) {
		col--
	}
	for col > 0 && !unicode.IsSpace(runes[col-1]) {
		col--
	}
	return col
}

func (m DetailModel) updateJumpPath(msg tea.KeyPressMsg) (DetailModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.jumpingPath = false
	case "enter":
		m.jumpingPath = false
		if m.jumpQuery != "" {
			m.jumpToPath(m.jumpQuery)
		}
	case "backspace":
		if len(m.jumpQuery) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.jumpQuery)
			m.jumpQuery = m.jumpQuery[:len(m.jumpQuery)-size]
		}
	default:
		if k := msg.String(); len([]rune(k)) == 1 {
			m.jumpQuery += k
		}
	}
	return m, nil
}

func (m *DetailModel) jumpToPath(target string) {
	var body string
	if m.mode == modeResponse && m.response != nil && len(m.response.Body) > 0 {
		body = string(m.response.Body)
	} else if m.mode == modeRequest && m.request != nil && m.request.Body != "" {
		body = m.request.Body
	}
	if body == "" {
		return
	}

	if m.activeTab() != 0 {
		m.setActiveTab(0)
	}

	content := m.currentTabContent()
	lines := strings.Split(content, "\n")
	target = strings.TrimSpace(target)
	if !strings.HasPrefix(target, "$") {
		target = "$." + target
	}
	for i := range lines {
		path := jsonLineToPath(body, i)
		if path == target || strings.HasSuffix(path, strings.TrimPrefix(target, "$")) {
			m.cursorLine = i
			m.cursorCol = 0
			m.scrollViewToCursor()
			return
		}
	}
}

func (m DetailModel) updateNormal(msg tea.KeyPressMsg) (DetailModel, tea.Cmd) {
	key := msg.String()

	// g-prefix
	if m.pendingG {
		m.pendingG = false
		switch key {
		case "g":
			m.cursorLine = 0
			m.cursorCol = 0
			m.scrollViewToCursor()
		case "p":
			m.jumpingPath = true
			m.jumpQuery = ""
		}
		return m, nil
	}

	// z-prefix (JSON fold controls)
	if m.pendingZ {
		m.pendingZ = false
		switch key {
		case "a":
			m.toggleFoldAtCursor()
		case "M":
			collapsed := m.collapsedFor()
			if lastFold.bodyStart >= 0 {
				for k := range lastFold.jsonFolds {
					collapsed[k] = true
				}
			}
		case "R":
			m.clearCollapsed()
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
	case "v":
		m.selecting = true
		m.selectLineMode = false
		m.selectAnchor = m.cursorLine
		m.selectAnchorCol = m.cursorCol
		m.selectCursor = m.cursorLine
		m.selectCol = m.cursorCol
		m.selectLines = m.activeTabRawLines()
		return m, nil
	case "V":
		m.selecting = true
		m.selectLineMode = true
		m.selectAnchor = m.cursorLine
		m.selectAnchorCol = 0
		m.selectCursor = m.cursorLine
		m.selectLines = m.activeTabRawLines()
		m.selectCol = m.selectLineLen(m.cursorLine)
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

	case "1":
		m.setActiveTab(0)
	case "2":
		if m.tabCount() > 1 {
			m.setActiveTab(1)
		}
	case "3":
		if m.tabCount() > 2 {
			m.setActiveTab(2)
		}
	case "4":
		if m.tabCount() > 3 {
			m.setActiveTab(3)
		}
	case " ", "space":
		next := (m.activeTab() + 1) % m.tabCount()
		m.setActiveTab(next)

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
				// Run pre-request script
				if loaded.PreRequestScript != "" {
					scriptCtx := &script.ScriptContext{
						Request: loaded,
						EnvVars: mergedVars,
					}
					if scriptErr := script.RunPreRequest(loaded.PreRequestScript, scriptCtx); scriptErr != nil {
						return responseReceived{err: fmt.Errorf("pre-request script: %w", scriptErr)}
					}
				}
				jar := cookies.JarForEnv(envName)
				resp, err := engine.ExecuteWithJar(loaded, jar)
				// Run post-response script
				if err == nil && loaded.PostResponseScript != "" {
					scriptCtx := &script.ScriptContext{
						Request:  loaded,
						Response: resp,
						EnvVars:  mergedVars,
					}
					if scriptErr := script.RunPostResponse(loaded.PostResponseScript, scriptCtx); scriptErr != nil {
						// Attach script error to response for display
						resp.ScriptError = scriptErr.Error()
					}
				}
				// Evaluate assertions against the resolved request so the
				// expected (RHS) values reflect interpolated variables.
				if err == nil && len(loaded.Assertions) > 0 {
					resp.AssertionResults = assert.EvaluateAll(loaded, resp)
				}
				return responseReceived{resp: resp, err: err}
			}
		}

	// Cursor movement
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "ctrl+d":
		m.moveCursor(m.viewableHeight() / 2)
	case "ctrl+u":
		m.moveCursor(-m.viewableHeight() / 2)
	case "g":
		m.pendingG = true
		return m, nil
	case "G":
		if lastFold.bodyStart >= 0 && len(lastFold.visible) > 0 {
			m.cursorLine = lastFold.visible[len(lastFold.visible)-1]
		} else {
			rawLines := m.activeTabRawLines()
			m.cursorLine = max(len(rawLines)-1, 0)
		}
		m.cursorCol = 0
		m.scrollViewToCursor()

	// Body controls
	case "w":
		m.wordWrap = !m.wordWrap
		m.clearCollapsed()
		m.setOffset(0)
	case "l":
		m.showLineNums = !m.showLineNums
	case "p":
		m.prettyPrint = !m.prettyPrint
		m.clearCollapsed()
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
	case "i":
		if m.mode == modeRequest && m.request != nil {
			focus := "body"
			if m.activeTab() == 1 {
				focus = "headers"
			}
			req := m.request
			return m, func() tea.Msg {
				return inlineEditRequest{request: req, focus: focus}
			}
		}
	}
	return m, nil
}



func (m DetailModel) diffView() string {
	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(colorText)
	sb.WriteString(title.Render("Response Diff") + "\n\n")

	if m.diffText == "" {
		sb.WriteString(dimStyle.Render("(no differences)"))
	} else {
		addStyle := lipgloss.NewStyle().Foreground(colorSuccess)
		removeStyle := lipgloss.NewStyle().Foreground(colorError)
		for _, line := range strings.Split(m.diffText, "\n") {
			if strings.HasPrefix(line, "+") {
				sb.WriteString(addStyle.Render(line) + "\n")
			} else if strings.HasPrefix(line, "-") {
				sb.WriteString(removeStyle.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	}

	sb.WriteString("\n" + dimStyle.Render("Esc: back to history"))
	return sb.String()
}

// --- Yank ---

func (m DetailModel) handleYank(key string) (DetailModel, tea.Cmd) {
	var text, label string

	switch key {
	case "l":
		text, label = m.yankCurrentLine()
	case "p":
		text, label = m.yankJSONPath()
	case "v":
		text, label = m.yankJSONValue()
	case "i":
		text, label = m.yankIndividualItem()
	default:
		if m.mode == modeRequest {
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
	}

	if text == "" {
		return m, nil
	}
	return m, copyCmd(text, label)
}

func (m DetailModel) yankCurrentLine() (string, string) {
	rawLines := m.activeTabRawLines()
	cur := m.cursorLine
	if cur < 0 || cur >= len(rawLines) {
		return "", ""
	}
	plain := strings.TrimSpace(rawLines[cur])
	if plain == "" {
		return "", ""
	}
	return plain, "line"
}

func (m DetailModel) yankJSONPath() (string, string) {
	var body string
	if m.mode == modeResponse && m.response != nil && len(m.response.Body) > 0 {
		body = string(m.response.Body)
	} else if m.mode == modeRequest && m.request != nil && m.request.Body != "" {
		body = m.request.Body
	}
	if body == "" {
		return "", ""
	}
	path := jsonLineToPath(body, m.cursorLine)
	if path == "$" {
		return "$", "JSON path"
	}
	return path, "JSON path"
}

func (m DetailModel) yankJSONValue() (string, string) {
	var body string
	if m.mode == modeResponse && m.response != nil && len(m.response.Body) > 0 {
		body = string(m.response.Body)
	} else if m.mode == modeRequest && m.request != nil && m.request.Body != "" {
		body = m.request.Body
	}
	if body == "" {
		return "", ""
	}
	path := jsonLineToPath(body, m.cursorLine)
	if path == "$" {
		return body, "JSON root"
	}
	gjsonPath := strings.TrimPrefix(path, "$.")
	gjsonPath = strings.ReplaceAll(gjsonPath, "[", ".")
	gjsonPath = strings.ReplaceAll(gjsonPath, "]", "")
	result := gjson.Get(body, gjsonPath)
	if !result.Exists() {
		return "", ""
	}
	val := result.String()
	if result.Type == gjson.JSON {
		val = result.Raw
	}
	return val, "JSON value"
}

func (m DetailModel) yankIndividualItem() (string, string) {
	if m.activeTab() == 1 {
		var headers []model.Header
		if m.mode == modeRequest && m.request != nil {
			headers = m.request.Headers
		} else if m.mode == modeResponse && m.response != nil {
			headers = m.response.Headers
		}
		rawLines := m.activeTabRawLines()
		cur := m.cursorLine
		if cur >= 0 && cur < len(rawLines) {
			plain := strings.TrimSpace(rawLines[cur])
			for _, h := range headers {
				if strings.Contains(plain, h.Key) && strings.Contains(plain, h.Value) {
					return h.Key + ": " + h.Value, "header " + h.Key
				}
			}
		}
	}
	return m.yankCurrentLine()
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

	return m, copyCmd(code, label)
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
	content := m.currentTabContent()
	lines := strings.Split(content, "\n")
	query := strings.ToLower(m.searchQuery)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.searchHits = append(m.searchHits, i)
		}
	}
}

func (m DetailModel) currentTabContent() string {
	return strings.Join(m.activeTabRawLines(), "\n")
}

// lastFold holds the fold layout from the most recent render so value-receiver
// scroll/toggle handlers can navigate visible lines (same pattern as the
// package-level section ranges).
var lastFold foldState

func (m DetailModel) collapsedFor() map[int]bool {
	if m.mode == modeRequest {
		return m.reqCollapsed
	}
	return m.respCollapsed
}

func (m DetailModel) computeTabFoldState(rawLines []string) foldState {
	fs := foldState{bodyStart: -1}
	if m.wordWrap || m.selecting || m.searching {
		return fs
	}
	if m.activeTab() != 0 {
		return fs
	}
	var body, ct string
	if m.mode == modeResponse && m.response != nil {
		body, ct = string(m.response.Body), m.response.ContentType
	} else if m.mode == modeRequest && m.request != nil {
		body, ct = m.request.Body, requestContentType(m.request)
	}
	if body == "" || len(body) > maxBodyDisplay || detectFormat(ct, body) != formatJSON || !gjson.Valid(body) {
		return fs
	}
	jf := jsonFolds(string(pretty.Pretty([]byte(body))))
	if len(jf) == 0 {
		return fs
	}
	fs.bodyStart = 0
	fs.jsonFolds = jf
	fs.visible = buildVisible(len(rawLines), 0, jf, m.collapsedFor())
	return fs
}

func foldSummary(lines []string, openFull, closeFull int) string {
	opener := strings.TrimRight(lines[openFull], " ")
	bracket := "}"
	if closeFull >= 0 && closeFull < len(lines) {
		if strings.HasPrefix(strings.TrimSpace(stripANSI(lines[closeFull])), "]") {
			bracket = "]"
		}
	}
	return opener + dimStyle.Render(fmt.Sprintf(" … %s (%d lines)", bracket, closeFull-openFull-1))
}

// withFoldMarker swaps a body line's leading indent for a gutter glyph so
// foldable JSON nodes are discoverable (▾ expanded, ▸ collapsed).
func withFoldMarker(line string, collapsed bool) string {
	glyph := "▾"
	if collapsed {
		glyph = "▸"
	}
	marker := lipgloss.NewStyle().Foreground(colorBorderActive).Render(glyph)
	if strings.HasPrefix(line, "  ") {
		return marker + " " + line[2:]
	}
	return marker + " " + line
}

// renderCursorLine draws the visual-mode cursor line: the selection span is
// highlighted and a block cursor marks the active column, so the moving end of
// the selection is visible.
func (m DetailModel) renderCursorLine(runes []rune, a, b int, sel bool) string {
	selStyle := lipgloss.NewStyle().Background(colorSelBg).Foreground(colorSelFg)
	curStyle := lipgloss.NewStyle().Background(colorText).Foreground(colorStatusBar)
	col := m.selectCol
	if col > len(runes) {
		col = len(runes)
	}
	var sb strings.Builder
	for i := 0; i < len(runes); i++ {
		ch := string(runes[i])
		switch {
		case i == col:
			sb.WriteString(curStyle.Render(ch))
		case sel && i >= a && i < b:
			sb.WriteString(selStyle.Render(ch))
		default:
			sb.WriteString(ch)
		}
	}
	if col >= len(runes) {
		sb.WriteString(curStyle.Render(" "))
	}
	return sb.String()
}

func (m *DetailModel) scrollViewToCursor() {
	vh := m.viewableHeight()
	vis := lastFold.visible
	if lastFold.bodyStart >= 0 && len(vis) > 0 {
		curIdx := visIndexOf(vis, m.cursorLine)
		offIdx := visIndexOf(vis, m.offset())
		if curIdx < offIdx {
			m.setOffset(vis[curIdx])
		} else if curIdx >= offIdx+vh {
			newOff := curIdx - vh + 1
			if newOff < 0 {
				newOff = 0
			}
			m.setOffset(vis[newOff])
		}
	} else {
		if m.cursorLine < m.offset() {
			m.setOffset(m.cursorLine)
		} else if m.cursorLine >= m.offset()+vh {
			m.setOffset(m.cursorLine - vh + 1)
		}
	}
}

func (m *DetailModel) moveCursor(n int) {
	rawLines := m.activeTabRawLines()
	maxLine := len(rawLines) - 1
	if maxLine < 0 {
		maxLine = 0
	}
	vis := lastFold.visible
	if lastFold.bodyStart >= 0 && len(vis) > 0 {
		cur := visIndexOf(vis, m.cursorLine) + n
		if cur < 0 {
			cur = 0
		}
		if cur > len(vis)-1 {
			cur = len(vis) - 1
		}
		m.cursorLine = vis[cur]
	} else {
		m.cursorLine += n
		if m.cursorLine < 0 {
			m.cursorLine = 0
		}
		if m.cursorLine > maxLine {
			m.cursorLine = maxLine
		}
	}
	m.scrollViewToCursor()
}

// scrollBy advances the offset by n lines, skipping JSON lines hidden inside
// collapsed folds (using the layout from the last render).
func (m *DetailModel) scrollBy(n int) {
	vis := lastFold.visible
	if lastFold.bodyStart < 0 || len(vis) == 0 {
		off := m.offset() + n
		if off < 0 {
			off = 0
		}
		m.setOffset(off)
		return
	}
	cur := visIndexOf(vis, m.offset()) + n
	if cur < 0 {
		cur = 0
	}
	if cur > len(vis)-1 {
		cur = len(vis) - 1
	}
	m.setOffset(vis[cur])
}

func (m *DetailModel) toggleFoldAtCursor() {
	if lastFold.bodyStart < 0 {
		return
	}
	j := m.cursorLine - lastFold.bodyStart
	if _, ok := lastFold.jsonFolds[j]; !ok {
		return
	}
	cm := m.collapsedFor()
	if cm[j] {
		delete(cm, j)
	} else {
		cm[j] = true
	}
}

func (m *DetailModel) clearCollapsed() {
	if m.mode == modeRequest {
		m.reqCollapsed = make(map[int]bool)
	} else {
		m.respCollapsed = make(map[int]bool)
	}
}

var lastGutterWidth int

func (m DetailModel) tabLabels() []string {
	if m.mode == modeRequest {
		hdrCount := 0
		if m.request != nil {
			hdrCount = len(m.request.Headers)
		}
		return []string{"Body", fmt.Sprintf("Headers (%d)", hdrCount), "Metadata"}
	}
	resp := m.response
	hdrCount := 0
	if resp != nil {
		hdrCount = len(resp.Headers)
	}
	timingLabel := "Timing"
	if resp != nil && resp.Timing.Total > 0 {
		timingLabel = fmt.Sprintf("Timing ── %s", formatDuration(resp.Timing.Total))
	}
	labels := []string{"Body", fmt.Sprintf("Headers (%d)", hdrCount), timingLabel}
	if resp != nil && len(resp.AssertionResults) > 0 {
		passed := assert.CountPassed(resp.AssertionResults)
		total := len(resp.AssertionResults)
		labels = append(labels, fmt.Sprintf("Assertions (%d/%d)", passed, total))
	}
	return labels
}

func (m DetailModel) sectionTabBar() string {
	active := m.activeTab()
	tabs := m.tabLabels()
	var parts []string
	for i, label := range tabs {
		if i == active {
			style := lipgloss.NewStyle().Bold(true).Foreground(colorText).Background(lipgloss.Color("#2A2A3C")).Padding(0, 1)
			parts = append(parts, zone.Mark(fmt.Sprintf("dt:tab:%d", i), style.Render(label)))
		} else {
			parts = append(parts, zone.Mark(fmt.Sprintf("dt:tab:%d", i), dimStyle.Padding(0, 1).Render(label)))
		}
	}
	return strings.Join(parts, dimStyle.Render(" "))
}

func (m DetailModel) activeTabRawLines() []string {
	switch m.activeTab() {
	case 0:
		return m.bodyRawLines()
	case 1:
		return m.headerRawLines()
	case 2:
		if m.mode == modeRequest {
			return m.metadataRawLines()
		}
		return m.timingRawLines()
	case 3:
		return m.assertionRawLines()
	}
	return nil
}

func (m DetailModel) bodyRawLines() []string {
	if m.mode == modeRequest {
		if m.request == nil || m.request.Body == "" {
			return []string{"(empty body)"}
		}
		body := m.request.Body
		if m.prettyPrint {
			body = formatBodyPlainPretty(body, requestContentType(m.request))
		}
		lines := strings.Split(body, "\n")
		if m.wordWrap {
			lines = wrapLines(lines, m.bodyWidth())
		}
		return lines
	}
	if m.response == nil || len(m.response.Body) == 0 {
		return []string{"(empty body)"}
	}
	if len(m.response.Body) > maxBodyDisplay {
		return []string{fmt.Sprintf("Response body is %s — too large. Use yb to copy.", formatSize(len(m.response.Body)))}
	}
	var body string
	if m.prettyPrint {
		body = formatBodyPlainPretty(string(m.response.Body), m.response.ContentType)
	} else {
		body = string(m.response.Body)
	}
	lines := strings.Split(body, "\n")
	if m.wordWrap {
		lines = wrapLines(lines, m.bodyWidth())
	}
	return lines
}

func (m DetailModel) headerRawLines() []string {
	var headers []model.Header
	if m.mode == modeRequest && m.request != nil {
		headers = m.request.Headers
	} else if m.mode == modeResponse && m.response != nil {
		headers = m.response.Headers
	}
	if len(headers) == 0 {
		return []string{"(no headers)"}
	}
	lines := make([]string, len(headers))
	for i, h := range headers {
		lines[i] = h.Key + ": " + h.Value
	}
	return lines
}

func (m DetailModel) metadataRawLines() []string {
	if m.request == nil {
		return []string{"(no metadata)"}
	}
	req := m.request
	var parts []string
	if req.Name != "" {
		parts = append(parts, "# @name "+req.Name)
	}
	if req.Metadata.NoRedirect {
		parts = append(parts, "# @no-redirect")
	}
	if req.Metadata.NoCookieJar {
		parts = append(parts, "# @no-cookie-jar")
	}
	if req.Metadata.Timeout > 0 {
		parts = append(parts, fmt.Sprintf("# @timeout %ds", int(req.Metadata.Timeout.Seconds())))
	}
	if req.Metadata.ConnTimeout > 0 {
		parts = append(parts, fmt.Sprintf("# @connection-timeout %ds", int(req.Metadata.ConnTimeout.Seconds())))
	}
	if len(parts) == 0 {
		return []string{"(no metadata)"}
	}
	return parts
}

func (m DetailModel) timingRawLines() []string {
	if m.response == nil || m.response.Timing.Total <= 0 {
		return []string{"(no timing data)"}
	}
	t := m.response.Timing
	phases := []struct {
		name string
		d    time.Duration
	}{
		{"DNS    ", t.DNS},
		{"Connect", t.Connect},
		{"TLS    ", t.TLS},
		{"TTFB   ", t.TTFB},
		{"Body   ", t.BodyRead},
		{"Total  ", t.Total},
	}
	lines := make([]string, len(phases))
	for i, p := range phases {
		lines[i] = fmt.Sprintf("%s  %s", p.name, formatDuration(p.d))
	}
	return lines
}

func (m DetailModel) assertionRawLines() []string {
	if m.response == nil || len(m.response.AssertionResults) == 0 {
		return []string{"(no assertions)"}
	}
	lines := make([]string, len(m.response.AssertionResults))
	for i, r := range m.response.AssertionResults {
		if r.Passed {
			lines[i] = "✓ " + r.Assertion.Raw
		} else {
			line := "✗ " + r.Assertion.Raw
			if r.Error != "" {
				line += fmt.Sprintf(" (%s)", r.Error)
			} else {
				line += fmt.Sprintf(" (got %s)", r.Actual)
			}
			lines[i] = line
		}
	}
	return lines
}

func formatBodyPlainPretty(body, contentType string) string {
	switch detectFormat(contentType, body) {
	case formatJSON:
		if gjson.Valid(body) {
			return string(pretty.Pretty([]byte(body)))
		}
	case formatXML:
		out := stripANSI(indentXML(body))
		if out != "" {
			return out
		}
	}
	return body
}

func (m DetailModel) activeTabColoredLines() []string {
	raw := m.activeTabRawLines()
	if m.activeTab() == 1 {
		keyStyle := lipgloss.NewStyle().Foreground(colorKey)
		colored := make([]string, len(raw))
		for i, line := range raw {
			if idx := strings.Index(line, ": "); idx > 0 {
				colored[i] = keyStyle.Render(line[:idx]) + ": " + line[idx+2:]
			} else {
				colored[i] = line
			}
		}
		return colored
	}
	if m.activeTab() == 3 {
		passStyle := lipgloss.NewStyle().Foreground(colorSuccess)
		failStyle := lipgloss.NewStyle().Foreground(colorError)
		colored := make([]string, len(raw))
		for i, line := range raw {
			if strings.HasPrefix(line, "✓") {
				colored[i] = passStyle.Render("✓") + line[len("✓"):]
			} else if strings.HasPrefix(line, "✗") {
				colored[i] = failStyle.Render("✗") + line[len("✗"):]
			} else {
				colored[i] = line
			}
		}
		return colored
	}
	if m.activeTab() == 2 && m.mode == modeResponse {
		return m.timingColoredLines()
	}
	if m.activeTab() != 0 || !m.prettyPrint {
		return raw
	}
	var body, ct string
	if m.mode == modeRequest && m.request != nil {
		body, ct = m.request.Body, requestContentType(m.request)
	} else if m.mode == modeResponse && m.response != nil {
		body, ct = string(m.response.Body), m.response.ContentType
	}
	if body == "" {
		return raw
	}
	colored := colorizeBody(body, ct)
	coloredLines := strings.Split(colored, "\n")
	if m.wordWrap {
		coloredLines = wrapLines(coloredLines, m.bodyWidth())
	}
	if len(coloredLines) != len(raw) {
		return raw
	}
	return coloredLines
}

func (m DetailModel) timingColoredLines() []string {
	if m.response == nil || m.response.Timing.Total <= 0 {
		return []string{dimStyle.Render("(no timing data)")}
	}
	t := m.response.Timing
	totalNs := t.Total.Nanoseconds()
	barWidth := 24
	phases := []struct {
		name string
		d    time.Duration
		clr  string
	}{
		{"DNS    ", t.DNS, "#89B4FA"},
		{"Connect", t.Connect, "#A6E3A1"},
		{"TLS    ", t.TLS, "#F9E2AF"},
		{"TTFB   ", t.TTFB, "#FAB387"},
		{"Body   ", t.BodyRead, "#CBA6F7"},
		{"Total  ", t.Total, "#CDD6F4"},
	}
	lines := make([]string, len(phases))
	for i, p := range phases {
		filled := int(int64(barWidth) * p.d.Nanoseconds() / totalNs)
		if p.d > 0 && filled == 0 {
			filled = 1
		}
		empty := barWidth - filled
		if empty < 0 {
			empty = 0
		}
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(p.clr)).Render(strings.Repeat("█", filled))
		bar += dimStyle.Render(strings.Repeat("░", empty))
		lines[i] = fmt.Sprintf("%s  %s  %s", dimStyle.Render(p.name), bar, formatDuration(p.d))
	}
	return lines
}

func (m DetailModel) View() string {
	if m.showDiff {
		return m.diffView()
	}
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
		sb.WriteString(lipgloss.NewStyle().Foreground(colorError).Render("✗ " + m.errMsg))
		sb.WriteString("\n\n")
	}

	if m.response != nil {
		sb.WriteString(m.toggleBar())
		sb.WriteString("\n")
	}

	if m.mode == modeResponse && m.response != nil {
		sb.WriteString(m.stickyStatus())
		if m.response.ScriptError != "" {
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().Foreground(colorWarning).Render("⚠ script: " + m.response.ScriptError))
		}
		if m.activeTab() == 0 && len(m.response.Body) > 0 && strings.Contains(strings.ToLower(m.response.ContentType), "json") {
			body := string(m.response.Body)
			path := jsonLineToPath(body, m.cursorLine)
			if path != "$" {
				sb.WriteString("\n")
				sb.WriteString(dimStyle.Render("  " + path))
			}
		}
		sb.WriteString("\n\n")
	} else if m.request != nil {
		method := lipgloss.NewStyle().Foreground(methodColor(m.request.Method)).Bold(true).Render(m.request.Method)
		sb.WriteString(fmt.Sprintf("%s %s\n\n", method, m.request.URL))
	}

	sb.WriteString(m.sectionTabBar())
	sb.WriteString("\n")

	rawLines := m.activeTabRawLines()
	coloredLines := m.activeTabColoredLines()

	off := m.offset()
	vh := m.viewableHeight()

	lastFold = m.computeTabFoldState(rawLines)

	gutterWidth := 0
	if m.showLineNums && m.activeTab() == 0 {
		gutterWidth = len(fmt.Sprintf("%d", len(rawLines))) + 3
	}
	lastGutterWidth = gutterWidth

	lineNumStyle := lipgloss.NewStyle().Foreground(colorLineNum)
	selectStyle := lipgloss.NewStyle().Background(colorSelBg).Foreground(colorSelFg)

	var visibleIndices []int
	if lastFold.bodyStart >= 0 {
		visibleIndices = buildVisible(len(rawLines), 0, lastFold.jsonFolds, m.collapsedFor())
	} else {
		visibleIndices = make([]int, len(rawLines))
		for i := range rawLines {
			visibleIndices[i] = i
		}
	}
	lastFold.visible = visibleIndices

	maxOff := len(visibleIndices) - vh
	if maxOff < 0 {
		maxOff = 0
	}
	offIdx := visIndexOf(visibleIndices, off)
	if offIdx > maxOff {
		offIdx = maxOff
	}
	if offIdx < 0 {
		offIdx = 0
	}
	if len(visibleIndices) > 0 {
		off = visibleIndices[offIdx]
		m.setOffset(off)
	}

	endIdx := offIdx + vh
	if endIdx > len(visibleIndices) {
		endIdx = len(visibleIndices)
	}

	lastVisibleRange = [2]int{0, 0}
	if endIdx > offIdx && len(visibleIndices) > 0 {
		lastVisibleRange = [2]int{visibleIndices[offIdx], visibleIndices[endIdx-1]}
	}

	collapsed := m.collapsedFor()
	for _, lineIdx := range visibleIndices[offIdx:endIdx] {
		var gutter string
		if gutterWidth > 0 {
			numW := gutterWidth - 3
			gutter = lineNumStyle.Render(fmt.Sprintf("%*d │ ", numW, lineIdx+1))
		}

		raw := rawLines[lineIdx]
		colored := raw
		if lineIdx < len(coloredLines) {
			colored = coloredLines[lineIdx]
		}

		if lastFold.bodyStart >= 0 {
			if c, ok := lastFold.jsonFolds[lineIdx]; ok {
				var foldLine string
				if collapsed[lineIdx] {
					foldLine = withFoldMarker(foldSummary(coloredLines, lineIdx, c), true)
				} else {
					foldLine = withFoldMarker(colored, false)
				}
				sb.WriteString(gutter + zone.Mark(fmt.Sprintf("fold:%d", lineIdx), foldLine) + "\n")
				continue
			}
		}

		if m.selecting {
			runes := []rune(raw)
			a, b, sel := m.lineSelCols(lineIdx, len(runes))
			if lineIdx == m.selectCursor {
				sb.WriteString(gutter + m.renderCursorLine(runes, a, b, sel) + "\n")
				continue
			}
			if sel {
				sb.WriteString(gutter + string(runes[:a]) + selectStyle.Render(string(runes[a:b])) + string(runes[b:]) + "\n")
				continue
			}
			sb.WriteString(gutter + colored + "\n")
			continue
		}

		content := colored
		if lineIdx == m.cursorLine {
			cursorLineStyle := lipgloss.NewStyle().Background(lipgloss.Color("#2A2A3C"))
			content = cursorLineStyle.Render(raw)
		}

		sb.WriteString(gutter + zone.Mark(fmt.Sprintf("dt:line:%d", lineIdx), content) + "\n")
	}

	if len(visibleIndices) > vh {
		pct := 0
		if maxOff > 0 {
			pct = offIdx * 100 / maxOff
		}
		sb.WriteString(dimStyle.Render(fmt.Sprintf("── %d%% (%d/%d) ──", pct, offIdx+1, len(visibleIndices))))
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
	if m.selecting {
		modeLabel := "VISUAL"
		if m.selectLineMode {
			modeLabel = "VISUAL LINE"
		}
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(colorBorderActive).Render(
			fmt.Sprintf("-- %s -- L%d:C%d  │  h/l/j/k/w/b:move  0/$:line  y:copy  esc:cancel", modeLabel, m.selectCursor+1, m.selectCol+1)))
	}
	if m.jumpingPath {
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(colorBorderActive).Render("path: "+m.jumpQuery+"█"))
	}
	if m.pendingG {
		sb.WriteString("\n" + dimStyle.Render("g- (g:top  p:jump to path)"))
	}
	if m.pendingZ {
		sb.WriteString("\n" + dimStyle.Render("z-"))
	}
	if m.pendingY {
		sb.WriteString("\n" + dimStyle.Render("y- (b:body  h:headers  a:all  c:curl  l:line  p:path  v:value  i:item  g:generate)"))
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
		reqStyle = lipgloss.NewStyle().Bold(true).Foreground(colorText).Underline(true)
	} else {
		respStyle = lipgloss.NewStyle().Bold(true).Foreground(colorText).Underline(true)
	}
	return zone.Mark("dt:req", reqStyle.Render("[r] Request")) + dimStyle.Render("  │  ") + zone.Mark("dt:resp", respStyle.Render("[s] Response"))
}



const maxBodyDisplay = 512 * 1024

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
		clr = colorSuccess
		icon = "✓"
	case code >= 300 && code < 400:
		clr = lipgloss.Color("#FFFF00")
		icon = "→"
	case code >= 400 && code < 500:
		clr = colorWarning
		icon = "✗"
	default:
		clr = colorError
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
		timing = formatDuration(resp.Timing.Total)
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

// --- History ---

func (m DetailModel) historyView() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorText).Bold(true).Render("Response History"))
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
			line = lipgloss.NewStyle().Foreground(colorWarning).Render("A " + line)
		} else if i == m.historyIdx {
			line = lipgloss.NewStyle().Background(colorSelBg).Foreground(colorSelFg).Render(line)
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
	return colorizeBody(string(resp.Body), resp.ContentType)
}

type bodyFormat int

const (
	formatPlain bodyFormat = iota
	formatJSON
	formatXML
)

// detectFormat picks a rendering format from the Content-Type, falling back to
// sniffing the first bytes. HTML is intentionally treated as plain because the
// XML tokenizer garbles real-world HTML (void/unclosed tags, entities).
func detectFormat(contentType, body string) bodyFormat {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json"):
		return formatJSON
	case strings.Contains(ct, "xml"):
		return formatXML
	case ct != "":
		return formatPlain
	}
	trimmed := strings.TrimSpace(body)
	switch {
	case strings.HasPrefix(trimmed, "{"), strings.HasPrefix(trimmed, "["):
		return formatJSON
	case strings.HasPrefix(trimmed, "<?xml"):
		return formatXML
	}
	return formatPlain
}

// colorizeBody pretty-prints and syntax-colors a JSON or XML body. Invalid JSON
// (e.g. an unresolved {{var}} template in a request) is returned unchanged so it
// is not mangled.
func colorizeBody(body, contentType string) string {
	if body == "" {
		return ""
	}
	switch detectFormat(contentType, body) {
	case formatJSON:
		if gjson.Valid(body) {
			return string(pretty.Color(pretty.Pretty([]byte(body)), nil))
		}
		return body
	case formatXML:
		return indentXML(body)
	default:
		return body
	}
}

func requestContentType(req *model.Request) string {
	for _, h := range req.Headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			return h.Value
		}
	}
	return ""
}

func indentXML(s string) string {
	decoder := xml.NewDecoder(strings.NewReader(s))
	var sb strings.Builder
	depth := 0
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	attrKeyStyle := lipgloss.NewStyle().Foreground(colorKey)
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

func formatDuration(d time.Duration) string {
	switch {
	case d <= 0:
		return "0ms"
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

func timingView(resp *model.Response) string {
	t := resp.Timing
	if t.Total <= 0 {
		return dimStyle.Render("  (no timing data)")
	}
	totalNs := t.Total.Nanoseconds()
	barWidth := 24
	phases := []struct {
		name string
		d    time.Duration
		clr  string
	}{
		{"DNS    ", t.DNS, "#89B4FA"},
		{"Connect", t.Connect, "#A6E3A1"},
		{"TLS    ", t.TLS, "#F9E2AF"},
		{"TTFB   ", t.TTFB, "#FAB387"},
		{"Body   ", t.BodyRead, "#CBA6F7"},
		{"Total  ", t.Total, "#CDD6F4"},
	}
	var sb strings.Builder
	for _, p := range phases {
		filled := int(int64(barWidth) * p.d.Nanoseconds() / totalNs)
		if p.d > 0 && filled == 0 {
			filled = 1
		}
		empty := barWidth - filled
		if empty < 0 {
			empty = 0
		}
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(p.clr)).Render(strings.Repeat("█", filled))
		bar += dimStyle.Render(strings.Repeat("░", empty))
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n", dimStyle.Render(p.name), bar, formatDuration(p.d)))
	}
	return sb.String()
}

// --- Utility ---

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

func formatSize(bytes int) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
