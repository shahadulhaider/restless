package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

type editorMode int

const (
	editorModeCreate editorMode = iota
	editorModeEdit
)

type focusedField int

const (
	fieldName focusedField = iota
	fieldMethod
	fieldURL
	fieldHeaderKey
	fieldHeaderValue
	fieldBody
	fieldNoRedirect
	fieldNoCookieJar
	fieldTimeout
	fieldCount
)

var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

type headerRow struct {
	key   lineEdit
	value lineEdit
}

// EditorModel is a form for creating or editing a model.Request.
type EditorModel struct {
	mode   editorMode
	focus  focusedField
	width  int
	height int

	name         lineEdit
	methodIdx    int
	url          lineEdit
	headers      []headerRow
	headerCursor int
	headerOnKey  bool
	body         []lineEdit // one lineEdit per body line
	bodyCursor   int
	noRedirect   bool
	noCookieJar  bool
	timeoutSecs  lineEdit
}

func NewEditorModel() EditorModel {
	return EditorModel{
		mode:    editorModeCreate,
		focus:   fieldName,
		name:    newLineEdit(""),
		url:     newLineEdit(""),
		headers: []headerRow{{key: newLineEdit(""), value: newLineEdit("")}},
		body:    []lineEdit{newLineEdit("")},
	}
}

func NewEditorModelFromRequest(req model.Request) EditorModel {
	m := EditorModel{
		mode:        editorModeEdit,
		focus:       fieldName,
		name:        newLineEdit(req.Name),
		url:         newLineEdit(req.URL),
		noRedirect:  req.Metadata.NoRedirect,
		noCookieJar: req.Metadata.NoCookieJar,
		headerOnKey: true,
	}
	m.methodIdx = 0
	for i, meth := range httpMethods {
		if meth == req.Method {
			m.methodIdx = i
			break
		}
	}
	if len(req.Headers) > 0 {
		for _, h := range req.Headers {
			m.headers = append(m.headers, headerRow{key: newLineEdit(h.Key), value: newLineEdit(h.Value)})
		}
	} else {
		m.headers = []headerRow{{key: newLineEdit(""), value: newLineEdit("")}}
	}
	if req.Body != "" {
		lines := strings.Split(req.Body, "\n")
		for _, l := range lines {
			m.body = append(m.body, newLineEdit(l))
		}
	} else {
		m.body = []lineEdit{newLineEdit("")}
	}
	if req.Metadata.Timeout > 0 {
		m.timeoutSecs = newLineEdit(strconv.Itoa(int(req.Metadata.Timeout.Seconds())))
	}
	return m
}

func (m EditorModel) Request() model.Request {
	req := model.Request{
		Name:   strings.TrimSpace(m.name.String()),
		Method: httpMethods[m.methodIdx],
		URL:    strings.TrimSpace(m.url.String()),
	}
	for _, h := range m.headers {
		k := strings.TrimSpace(h.key.String())
		v := strings.TrimSpace(h.value.String())
		if k != "" {
			req.Headers = append(req.Headers, model.Header{Key: k, Value: v})
		}
	}
	var bodyLines []string
	for _, l := range m.body {
		bodyLines = append(bodyLines, l.String())
	}
	req.Body = strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")
	req.Metadata.NoRedirect = m.noRedirect
	req.Metadata.NoCookieJar = m.noCookieJar
	if secs, err := strconv.Atoi(strings.TrimSpace(m.timeoutSecs.String())); err == nil && secs > 0 {
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

		switch m.focus {
		case fieldMethod:
			switch key {
			case "left":
				m.methodIdx = (m.methodIdx - 1 + len(httpMethods)) % len(httpMethods)
			case "right":
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
			m.name.HandleKey(key)
		case fieldURL:
			m.url.HandleKey(key)
		case fieldTimeout:
			m.timeoutSecs.HandleKeyFiltered(key, isDigit)
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
	case "up":
		if m.headerCursor > 0 {
			m.headerCursor--
		}
	case "down":
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
			m.headers[0] = headerRow{key: newLineEdit(""), value: newLineEdit("")}
		}
	case "enter":
		m.focus = fieldHeaderValue
		m.headerOnKey = false
	default:
		m.headers[m.headerCursor].key.HandleKey(key)
	}
	return m
}

func (m EditorModel) handleHeaderValue(key string) EditorModel {
	switch key {
	case "enter":
		m.headers = append(m.headers, headerRow{key: newLineEdit(""), value: newLineEdit("")})
		m.headerCursor = len(m.headers) - 1
		m.focus = fieldHeaderKey
		m.headerOnKey = true
	case "up":
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
			m.headers[0] = headerRow{key: newLineEdit(""), value: newLineEdit("")}
		}
		m.focus = fieldHeaderKey
		m.headerOnKey = true
	default:
		m.headers[m.headerCursor].value.HandleKey(key)
	}
	return m
}

func (m EditorModel) handleBody(key string) EditorModel {
	switch key {
	case "up":
		if m.bodyCursor > 0 {
			m.bodyCursor--
			// Clamp cursor position to new line length
			if m.body[m.bodyCursor].pos > m.body[m.bodyCursor].Len() {
				m.body[m.bodyCursor].pos = m.body[m.bodyCursor].Len()
			}
		}
	case "down":
		if m.bodyCursor < len(m.body)-1 {
			m.bodyCursor++
			if m.body[m.bodyCursor].pos > m.body[m.bodyCursor].Len() {
				m.body[m.bodyCursor].pos = m.body[m.bodyCursor].Len()
			}
		}
	case "enter":
		// Split current line at cursor position
		cur := &m.body[m.bodyCursor]
		after := string(cur.text[cur.pos:])
		cur.text = cur.text[:cur.pos]
		// Insert new line after current
		newLine := newLineEdit(after)
		newLine.pos = 0
		m.body = append(m.body, lineEdit{})
		copy(m.body[m.bodyCursor+2:], m.body[m.bodyCursor+1:])
		m.body[m.bodyCursor+1] = newLine
		m.bodyCursor++
	case "backspace":
		cur := &m.body[m.bodyCursor]
		if cur.pos > 0 {
			cur.Backspace()
		} else if m.bodyCursor > 0 {
			// Merge with previous line
			prev := &m.body[m.bodyCursor-1]
			joinPos := prev.Len()
			prev.text = append(prev.text, cur.text...)
			prev.pos = joinPos
			m.body = append(m.body[:m.bodyCursor], m.body[m.bodyCursor+1:]...)
			m.bodyCursor--
		}
	default:
		m.body[m.bodyCursor].HandleKey(key)
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

// --- View ---

func (m EditorModel) View() string {
	var sb strings.Builder

	title := "New Request"
	if m.mode == editorModeEdit {
		title = "Edit Request"
	}
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4")).Render(title))
	sb.WriteString("\n\n")

	sb.WriteString(m.renderLineEditField("Name", &m.name, m.focus == fieldName))
	sb.WriteString(m.renderMethodField())
	sb.WriteString(m.renderLineEditField("URL", &m.url, m.focus == fieldURL))
	sb.WriteString(m.renderHeadersField())
	sb.WriteString(m.renderBodyField())
	sb.WriteString(m.renderToggleField("@no-redirect", m.noRedirect, m.focus == fieldNoRedirect))
	sb.WriteString(m.renderToggleField("@no-cookie-jar", m.noCookieJar, m.focus == fieldNoCookieJar))
	sb.WriteString(m.renderLineEditField("@timeout (sec)", &m.timeoutSecs, m.focus == fieldTimeout))

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("Tab: next  Ctrl+S: save  Esc: cancel  ←/→: move  Ctrl+A/E: home/end  Ctrl+W: del word  Ctrl+U: clear"))

	return sb.String()
}

func (m EditorModel) renderLineEditField(label string, le *lineEdit, focused bool) string {
	indicator := "  "
	labelStyle := dimStyle
	if focused {
		indicator = "> "
		labelStyle = lipgloss.NewStyle().Foreground(colorBorderActive)
	}
	return fmt.Sprintf("%s%s: %s\n", indicator, labelStyle.Render(label), le.View(focused))
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
		rowIndicator := "    "
		if isActive {
			rowIndicator = "  > "
		}
		keyView := h.key.View(isActive && m.focus == fieldHeaderKey)
		valView := h.value.View(isActive && m.focus == fieldHeaderValue)
		sb.WriteString(fmt.Sprintf("%s%s: %s\n", rowIndicator, keyView, valView))
	}
	if headerFocused {
		sb.WriteString(dimStyle.Render("    Enter: value/add row  Ctrl+D: del row  ↑/↓: rows") + "\n")
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
	for i := range m.body {
		isCurrent := focused && i == m.bodyCursor
		sb.WriteString(fmt.Sprintf("    %s\n", m.body[i].View(isCurrent)))
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

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
