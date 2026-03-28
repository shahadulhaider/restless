package tui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tidwall/pretty"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/model"
)

type detailTab int

const (
	tabHeaders detailTab = iota
	tabBody
	tabTiming
)

type DetailModel struct {
	request  *model.Request
	response *model.Response
	tab      detailTab
	sending  bool
	width    int
	height   int
	offset   int
}

type responseReceived struct {
	resp *model.Response
	err  error
}

func NewDetailModel() DetailModel {
	return DetailModel{}
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) SetRequest(req *model.Request) DetailModel {
	m.request = req
	m.response = nil
	m.offset = 0
	return m
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case RequestSelected:
		m.request = msg.Request
		m.response = nil
		m.offset = 0

	case responseReceived:
		m.sending = false
		if msg.err == nil {
			m.response = msg.resp
		}
		m.offset = 0

	case tea.KeyPressMsg:
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
		case "enter", "ctrl+r":
			if m.request != nil && !m.sending {
				m.sending = true
				req := m.request
				return m, func() tea.Msg {
					resp, err := engine.Execute(req)
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
	if m.request == nil {
		return dimStyle.Render("Request / Response\n\n(select a request to view)")
	}

	var sb strings.Builder

	if m.sending {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render("Sending..."))
		sb.WriteString("\n\n")
	}

	if m.response == nil {
		sb.WriteString(requestView(m.request))
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Enter / ctrl+r to send"))
		return sb.String()
	}

	tabs := []string{"1:Headers", "2:Body", "3:Timing"}
	var tabBar strings.Builder
	for i, t := range tabs {
		if detailTab(i) == m.tab {
			tabBar.WriteString(lipgloss.NewStyle().
				Underline(true).
				Foreground(lipgloss.Color("#CDD6F4")).
				Render(t))
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
