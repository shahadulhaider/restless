package tui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

type ItemType int

const (
	ItemTypeDir ItemType = iota
	ItemTypeFile
	ItemTypeRequest
)

type BrowserItem struct {
	Type    ItemType
	Depth   int
	Label   string
	Path    string
	Request *model.Request
}

type RequestSelected struct {
	Request *model.Request
}

type BrowserModel struct {
	collection    *model.Collection
	items         []BrowserItem
	cursor        int
	expanded      map[string]bool
	selected      *model.Request
	height        int
	width         int
	offset        int
	lastStatus    map[string]int // request key → last HTTP status code
}

func NewBrowserModel() BrowserModel {
	return BrowserModel{expanded: make(map[string]bool), lastStatus: make(map[string]int)}
}

func LoadCollection(rootDir string) (*model.Collection, error) {
	c := &model.Collection{RootDir: rootDir}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".http") {
			return nil
		}
		reqs, parseErr := parser.ParseFile(path)
		if parseErr != nil {
			return nil
		}
		c.Files = append(c.Files, model.HTTPFile{Path: path, Requests: reqs})
		return nil
	})

	sort.Slice(c.Files, func(i, j int) bool {
		return c.Files[i].Path < c.Files[j].Path
	})

	return c, err
}

func (m *BrowserModel) SetCollection(c *model.Collection) {
	m.collection = c
	m.items = m.buildItems()
}

func (m BrowserModel) buildItems() []BrowserItem {
	if m.collection == nil {
		return nil
	}

	type dirGroup struct {
		dir   string
		files []model.HTTPFile
	}
	groups := map[string]*dirGroup{}
	var order []string

	for _, f := range m.collection.Files {
		rel, _ := filepath.Rel(m.collection.RootDir, filepath.Dir(f.Path))
		if rel == "" || rel == "." {
			rel = "."
		}
		if _, ok := groups[rel]; !ok {
			groups[rel] = &dirGroup{dir: rel}
			order = append(order, rel)
		}
		groups[rel].files = append(groups[rel].files, f)
	}
	sort.Strings(order)

	var items []BrowserItem
	for _, dir := range order {
		g := groups[dir]
		if dir == "." {
			for _, f := range g.files {
				items = append(items, BrowserItem{
					Type:  ItemTypeFile,
					Depth: 0,
					Label: filepath.Base(f.Path),
					Path:  f.Path,
				})
				if m.expanded[f.Path] {
					for i := range f.Requests {
						req := &f.Requests[i]
						items = append(items, BrowserItem{
							Type:    ItemTypeRequest,
							Depth:   1,
							Label:   requestLabel(req),
							Path:    f.Path,
							Request: req,
						})
					}
				}
			}
		} else {
			icon := "▶"
			if m.expanded[dir] {
				icon = "▼"
			}
			items = append(items, BrowserItem{
				Type:  ItemTypeDir,
				Depth: 0,
				Label: icon + " " + dir + "/",
				Path:  dir,
			})
			if m.expanded[dir] {
				sort.Slice(g.files, func(i, j int) bool {
					return g.files[i].Path < g.files[j].Path
				})
				for _, f := range g.files {
					items = append(items, BrowserItem{
						Type:  ItemTypeFile,
						Depth: 1,
						Label: filepath.Base(f.Path),
						Path:  f.Path,
					})
					if m.expanded[f.Path] {
						for i := range f.Requests {
							req := &f.Requests[i]
							items = append(items, BrowserItem{
								Type:    ItemTypeRequest,
								Depth:   2,
								Label:   requestLabel(req),
								Path:    f.Path,
								Request: req,
							})
						}
					}
				}
			}
		}
	}
	return items
}

func requestLabel(req *model.Request) string {
	if req.Name != "" {
		return req.Name
	}
	return req.Method + " " + req.URL
}

// RecordStatus stores the last response status for a request.
func (m *BrowserModel) RecordStatus(req *model.Request, statusCode int) {
	if req != nil {
		m.lastStatus[requestKey(req)] = statusCode
	}
}

func requestKey(req *model.Request) string {
	return req.SourceFile + ":" + fmt.Sprintf("%d", req.SourceLine)
}

// CurrentItem returns the BrowserItem under the cursor, or nil.
func (m *BrowserModel) CurrentItem() *BrowserItem {
	if m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

func (m BrowserModel) Init() tea.Cmd {
	return nil
}

func (m BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.height {
					m.offset = m.cursor - m.height + 1
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "enter":
			if m.cursor < len(m.items) {
				item := m.items[m.cursor]
				switch item.Type {
				case ItemTypeDir:
					m.expanded[item.Path] = !m.expanded[item.Path]
					m.items = m.buildItems()
				case ItemTypeFile:
					m.expanded[item.Path] = !m.expanded[item.Path]
					m.items = m.buildItems()
				case ItemTypeRequest:
					m.selected = item.Request
					return m, func() tea.Msg {
						return RequestSelected{Request: item.Request}
					}
				}
			}
		}
	}
	return m, nil
}

func (m BrowserModel) View() string {
	if len(m.items) == 0 {
		return dimStyle.Render("Collection Browser\n\n(no requests loaded)")
	}

	var sb strings.Builder

	// Collection stats
	if m.collection != nil {
		reqCount := 0
		for _, f := range m.collection.Files {
			reqCount += len(f.Requests)
		}
		sb.WriteString(dimStyle.Render(fmt.Sprintf("%d reqs │ %d files", reqCount, len(m.collection.Files))))
		sb.WriteString("\n")
	}
	end := m.offset + m.height
	if end > len(m.items) {
		end = len(m.items)
	}
	start := m.offset
	if start > len(m.items) {
		start = len(m.items)
	}

	for i := start; i < end; i++ {
		item := m.items[i]
		indent := strings.Repeat("  ", item.Depth)
		var line string

		switch item.Type {
		case ItemTypeDir:
			line = indent + item.Label
		case ItemTypeFile:
			line = indent + "📄 " + item.Label
		case ItemTypeRequest:
			method := item.Request.Method
			colored := lipgloss.NewStyle().Foreground(methodColor(method)).Render(method)
			urlPart := " " + item.Request.URL
			if item.Request.Name != "" {
				urlPart = " " + item.Request.Name
			}
			// Status indicator from last response
			statusDot := ""
			if code, ok := m.lastStatus[requestKey(item.Request)]; ok {
				if code >= 200 && code < 300 {
					statusDot = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50")).Render("● ")
				} else if code >= 400 {
					statusDot = lipgloss.NewStyle().Foreground(lipgloss.Color("#F44336")).Render("● ")
				} else {
					statusDot = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render("● ")
				}
			}
			line = indent + statusDot + colored + urlPart
		}

		if i == m.cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#3D3D5C")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Render(fmt.Sprintf("%-*s", m.width-item.Depth*2, line))
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func methodColor(method string) color.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return lipgloss.Color("#4CAF50")
	case "POST":
		return lipgloss.Color("#2196F3")
	case "PUT":
		return lipgloss.Color("#FF9800")
	case "DELETE":
		return lipgloss.Color("#F44336")
	case "PATCH":
		return lipgloss.Color("#FF5722")
	default:
		return colorDim
	}
}
