package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/model"
)

// EditorSaved is emitted when the user saves the editor form.
type EditorSaved struct {
	Request model.Request
}

// EditorCancelled is emitted when the user cancels the editor form.
type EditorCancelled struct{}

// editorMode distinguishes create vs edit.
type editorMode int

const (
	editorModeCreate editorMode = iota
	editorModeEdit
)

// focusedField identifies which form field has focus.
type focusedField int

const (
	fieldName focusedField = iota
	fieldMethod
	fieldURL
	fieldHeaderKey  // header sub-focus: key column of current row
	fieldHeaderValue // header sub-focus: value column of current row
	fieldBody
	fieldNoRedirect
	fieldNoCookieJar
	fieldTimeout
	fieldCount // sentinel
)

var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

// headerRow is a mutable key-value pair for the headers list.
type headerRow struct {
	key   string
	value string
}

// EditorModel is a full-screen form for creating or editing a model.Request.
type EditorModel struct {
	mode   editorMode
	focus  focusedField
	width  int
	height int

	// Field values
	name         string
	methodIdx    int // index into httpMethods
	url          string
	headers      []headerRow
	headerCursor int // which header row is active
	headerOnKey  bool // true = cursor on key, false = on value
	body         []string // lines
	bodyCursor   int // active line in body
	noRedirect   bool
	noCookieJar  bool
	timeoutSecs  string
}

// NewEditorModel creates an empty editor in Create mode.
func NewEditorModel() EditorModel {
	return EditorModel{
		mode:    editorModeCreate,
		focus:   fieldName,
		headers: []headerRow{{}}, // start with one empty row
		body:    []string{""},
	}
}

// NewEditorModelFromRequest creates an editor pre-populated from req (Edit mode).
func NewEditorModelFromRequest(req model.Request) EditorModel {
	m := EditorModel{
		mode:        editorModeEdit,
		focus:       fieldName,
		name:        req.Name,
		url:         req.URL,
		noRedirect:  req.Metadata.NoRedirect,
		noCookieJar: req.Metadata.NoCookieJar,
		headerOnKey: true,
	}
	// Method
	m.methodIdx = 0
	for i, meth := range httpMethods {
		if meth == req.Method {
			m.methodIdx = i
			break
		}
	}
	// Headers
	if len(req.Headers) > 0 {
		for _, h := range req.Headers {
			m.headers = append(m.headers, headerRow{key: h.Key, value: h.Value})
		}
	} else {
		m.headers = []headerRow{{}}
	}
	// Body
	if req.Body != "" {
		m.body = strings.Split(req.Body, "\n")
	} else {
		m.body = []string{""}
	}
	// Timeout
	if req.Metadata.Timeout > 0 {
		m.timeoutSecs = strconv.Itoa(int(req.Metadata.Timeout.Seconds()))
	}
	return m
}

// Request returns the model.Request represented by the current form state.
func (m EditorModel) Request() model.Request {
	req := model.Request{
		Name:   strings.TrimSpace(m.name),
		Method: httpMethods[m.methodIdx],
		URL:    strings.TrimSpace(m.url),
	}
	for _, h := range m.headers {
		k := strings.TrimSpace(h.key)
		v := strings.TrimSpace(h.value)
		if k != "" {
			req.Headers = append(req.Headers, model.Header{Key: k, Value: v})
		}
	}
	bodyStr := strings.Join(m.body, "\n")
	req.Body = strings.TrimRight(bodyStr, "\n")
	req.Metadata.NoRedirect = m.noRedirect
	req.Metadata.NoCookieJar = m.noCookieJar
	if secs, err := strconv.Atoi(strings.TrimSpace(m.timeoutSecs)); err == nil && secs > 0 {
		req.Metadata.Timeout = time.Duration(secs) * time.Second
	}
	return req
}

func (m EditorModel) Init() tea.Cmd { return nil }

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		key := msg.String()

		// Global shortcuts
		switch key {
		case "ctrl+s":
			req := m.Request()
			return m, func() tea.Msg { return EditorSaved{Request: req} }
		case "esc":
			return m, func() tea.Msg { return EditorCancelled{} }
		case "tab":
			m = m.focusNext()
			return m, nil
		case "shift+tab":
			m = m.focusPrev()
			return m, nil
		}

		// Field-specific handling
		switch m.focus {
		case fieldMethod:
			switch key {
			case "left", "h":
				m.methodIdx = (m.methodIdx - 1 + len(httpMethods)) % len(httpMethods)
			case "right", "l":
				m.methodIdx = (m.methodIdx + 1) % len(httpMethods)
			}
		case fieldNoRedirect:
			if key == " " || key == "enter" {
				m.noRedirect = !m.noRedirect
			}
		case fieldNoCookieJar:
			if key == " " || key == "enter" {
				m.noCookieJar = !m.noCookieJar
			}
		case fieldName:
			m.name = editText(m.name, key)
		case fieldURL:
			m.url = editText(m.url, key)
		case fieldTimeout:
			m.timeoutSecs = editTextFiltered(m.timeoutSecs, key, isDigit)
		case fieldHeaderKey:
			m = m.handleHeaderKey(key)
		case fieldHeaderValue:
			m = m.handleHeaderValue(key)
		case fieldBody:
			m = m.handleBody(key)
		}
	}
	return m, nil
}

func (m EditorModel) handleHeaderKey(key string) EditorModel {
	switch key {
	case "up", "k":
		if m.headerCursor > 0 {
			m.headerCursor--
		}
	case "down", "j":
		if m.headerCursor < len(m.headers)-1 {
			m.headerCursor++
		}
	case "ctrl+d":
		if len(m.headers) > 1 {
			m.headers = append(m.headers[:m.headerCursor], m.headers[m.headerCursor+1:]...)
			if m.headerCursor >= len(m.headers) {
				m.headerCursor = len(m.headers) - 1
			}
		} else {
			m.headers[0] = headerRow{}
		}
	case "enter":
		// Move to value column
		m.focus = fieldHeaderValue
		m.headerOnKey = false
	default:
		m.headers[m.headerCursor].key = editText(m.headers[m.headerCursor].key, key)
	}
	return m
}

func (m EditorModel) handleHeaderValue(key string) EditorModel {
	switch key {
	case "enter":
		// Add a new empty row and move to its key
		m.headers = append(m.headers, headerRow{})
		m.headerCursor = len(m.headers) - 1
		m.focus = fieldHeaderKey
		m.headerOnKey = true
	case "up", "k":
		if m.headerCursor > 0 {
			m.headerCursor--
			m.focus = fieldHeaderKey
			m.headerOnKey = true
		}
	case "ctrl+d":
		if len(m.headers) > 1 {
			m.headers = append(m.headers[:m.headerCursor], m.headers[m.headerCursor+1:]...)
			if m.headerCursor >= len(m.headers) {
				m.headerCursor = len(m.headers) - 1
			}
		} else {
			m.headers[0] = headerRow{}
		}
		m.focus = fieldHeaderKey
		m.headerOnKey = true
	default:
		m.headers[m.headerCursor].value = editText(m.headers[m.headerCursor].value, key)
	}
	return m
}

func (m EditorModel) handleBody(key string) EditorModel {
	switch key {
	case "up":
		if m.bodyCursor > 0 {
			m.bodyCursor--
		}
	case "down":
		if m.bodyCursor < len(m.body)-1 {
			m.bodyCursor++
		}
	case "enter":
		// Insert new line after current
		newLines := make([]string, len(m.body)+1)
		copy(newLines, m.body[:m.bodyCursor+1])
		newLines[m.bodyCursor+1] = ""
		copy(newLines[m.bodyCursor+2:], m.body[m.bodyCursor+1:])
		m.body = newLines
		m.bodyCursor++
	case "backspace":
		if len(m.body[m.bodyCursor]) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.body[m.bodyCursor])
			m.body[m.bodyCursor] = m.body[m.bodyCursor][:len(m.body[m.bodyCursor])-size]
		} else if m.bodyCursor > 0 {
			// Merge with previous line
			m.body[m.bodyCursor-1] += m.body[m.bodyCursor]
			m.body = append(m.body[:m.bodyCursor], m.body[m.bodyCursor+1:]...)
			m.bodyCursor--
		}
	default:
		if len(key) == 1 || (len(key) > 1 && !strings.HasPrefix(key, "ctrl") && !strings.HasPrefix(key, "alt")) {
			m.body[m.bodyCursor] += key
		}
	}
	return m
}

func (m EditorModel) focusNext() EditorModel {
	switch m.focus {
	case fieldHeaderKey:
		m.focus = fieldHeaderValue
		m.headerOnKey = false
	case fieldHeaderValue:
		m.focus = fieldBody
		m.headerOnKey = true
	default:
		next := m.focus + 1
		if next >= fieldCount {
			next = 0
		}
		m.focus = next
		if m.focus == fieldHeaderKey {
			m.headerOnKey = true
		}
	}
	return m
}

func (m EditorModel) focusPrev() EditorModel {
	switch m.focus {
	case fieldHeaderValue:
		m.focus = fieldHeaderKey
		m.headerOnKey = true
	case fieldBody:
		m.focus = fieldHeaderValue
		m.headerOnKey = false
	default:
		prev := m.focus - 1
		if prev < 0 {
			prev = fieldCount - 1
		}
		m.focus = prev
	}
	return m
}

func (m EditorModel) View() string {
	var sb strings.Builder

	title := "New Request"
	if m.mode == editorModeEdit {
		title = "Edit Request"
	}
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4")).Render(title))
	sb.WriteString("\n\n")

	sb.WriteString(m.renderTextField("Name", m.name, m.focus == fieldName))
	sb.WriteString(m.renderMethodField())
	sb.WriteString(m.renderTextField("URL", m.url, m.focus == fieldURL))
	sb.WriteString(m.renderHeadersField())
	sb.WriteString(m.renderBodyField())
	sb.WriteString(m.renderToggleField("@no-redirect", m.noRedirect, m.focus == fieldNoRedirect))
	sb.WriteString(m.renderToggleField("@no-cookie-jar", m.noCookieJar, m.focus == fieldNoCookieJar))
	sb.WriteString(m.renderTextField("@timeout (seconds)", m.timeoutSecs, m.focus == fieldTimeout))

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("Tab/Shift+Tab: navigate  Ctrl+S: save  Esc: cancel"))

	return sb.String()
}

func (m EditorModel) renderTextField(label, value string, focused bool) string {
	indicator := "  "
	labelStyle := dimStyle
	if focused {
		indicator = "> "
		labelStyle = lipgloss.NewStyle().Foreground(colorBorderActive)
	}
	cursor := ""
	if focused {
		cursor = "█"
	}
	return fmt.Sprintf("%s%s: %s%s\n", indicator, labelStyle.Render(label), value, cursor)
}

func (m EditorModel) renderMethodField() string {
	focused := m.focus == fieldMethod
	indicator := "  "
	labelStyle := dimStyle
	if focused {
		indicator = "> "
		labelStyle = lipgloss.NewStyle().Foreground(colorBorderActive)
	}
	hint := ""
	if focused {
		hint = dimStyle.Render(" (←/→ to change)")
	}
	method := lipgloss.NewStyle().Foreground(methodColor(httpMethods[m.methodIdx])).Bold(true).Render(httpMethods[m.methodIdx])
	return fmt.Sprintf("%s%s: %s%s\n", indicator, labelStyle.Render("Method"), method, hint)
}

func (m EditorModel) renderHeadersField() string {
	var sb strings.Builder
	headerFocused := m.focus == fieldHeaderKey || m.focus == fieldHeaderValue
	indicator := "  "
	labelStyle := dimStyle
	if headerFocused {
		indicator = "> "
		labelStyle = lipgloss.NewStyle().Foreground(colorBorderActive)
	}
	sb.WriteString(fmt.Sprintf("%s%s:\n", indicator, labelStyle.Render("Headers")))

	for i, h := range m.headers {
		isActive := headerFocused && i == m.headerCursor
		keyCursor := ""
		valCursor := ""
		rowIndicator := "    "
		if isActive {
			rowIndicator = "  ● "
			if m.focus == fieldHeaderKey {
				keyCursor = "█"
			} else {
				valCursor = "█"
			}
		}
		line := fmt.Sprintf("%s%s%s: %s%s\n", rowIndicator, h.key, keyCursor, h.value, valCursor)
		sb.WriteString(line)
	}
	if headerFocused {
		sb.WriteString(dimStyle.Render("    Enter: edit value / add row  Ctrl+D: delete row") + "\n")
	}
	return sb.String()
}

func (m EditorModel) renderBodyField() string {
	focused := m.focus == fieldBody
	indicator := "  "
	labelStyle := dimStyle
	if focused {
		indicator = "> "
		labelStyle = lipgloss.NewStyle().Foreground(colorBorderActive)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s%s:\n", indicator, labelStyle.Render("Body")))
	for i, line := range m.body {
		cursor := ""
		if focused && i == m.bodyCursor {
			cursor = "█"
		}
		sb.WriteString(fmt.Sprintf("    %s%s\n", line, cursor))
	}
	return sb.String()
}

func (m EditorModel) renderToggleField(label string, value, focused bool) string {
	indicator := "  "
	labelStyle := dimStyle
	if focused {
		indicator = "> "
		labelStyle = lipgloss.NewStyle().Foreground(colorBorderActive)
	}
	check := "[ ]"
	if value {
		check = "[x]"
	}
	return fmt.Sprintf("%s%s %s\n", indicator, labelStyle.Render(label), check)
}

// editText appends/removes characters for a single-line text field.
func editText(s, key string) string {
	switch key {
	case "backspace":
		if len(s) == 0 {
			return s
		}
		_, size := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-size]
	default:
		// Only accept printable single-character keys
		r := []rune(key)
		if len(r) == 1 {
			return s + key
		}
		return s
	}
}

// editTextFiltered is like editText but only allows characters matching filter.
func editTextFiltered(s, key string, filter func(rune) bool) string {
	switch key {
	case "backspace":
		if len(s) == 0 {
			return s
		}
		_, size := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-size]
	default:
		r := []rune(key)
		if len(r) == 1 && filter(r[0]) {
			return s + key
		}
		return s
	}
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
