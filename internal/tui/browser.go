package tui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

// ItemType distinguishes tree node types.
type ItemType int

const (
	ItemTypeDir ItemType = iota
	ItemTypeFile
	ItemTypeRequest
)

// BrowserItem is a flattened tree entry.
type BrowserItem struct {
	Type    ItemType
	Depth   int
	Label   string
	Path    string
	Request *model.Request
}

// RequestSelected is sent when the user presses Enter on a request.
type RequestSelected struct {
	Request *model.Request
}

// BrowserModel is the left-pane collection tree.
type BrowserModel struct {
	collection *model.Collection
	items      []BrowserItem
	cursor     int
	expanded   map[string]bool
	selected   *model.Request
	height     int
	width      int
	offset     int
}

func NewBrowserModel() BrowserModel {
	return BrowserModel{expanded: make(map[string]bool)}
}

// LoadCollection walks rootDir, parses .http files, and returns a Collection.
func LoadCollection(rootDir string) (*model.Collection, error) {
	c := &model.Collection{RootDir: rootDir}

	// Collect all .http files
	type fileEntry struct {
		relDir  string
		absPath string
	}
	var files []fileEntry

	entries, err := filepath.Glob(filepath.Join(rootDir, "**/*.http"))
	if err != nil {
		return c, nil
	}
	// filepath.Glob doesn't recurse; use a manual walk
	_ = entries

	var walkFiles func(dir string) error
	walkFiles = func(dir string) error {
		dirEntries, err := filepath.Glob(filepath.Join(dir, "*.http"))
		if err == nil {
			for _, f := range dirEntries {
				rel, _ := filepath.Rel(rootDir, filepath.Dir(f))
				files = append(files, fileEntry{relDir: rel, absPath: f})
			}
		}
		subdirs, _ := filepath.Glob(filepath.Join(dir, "*"))
		for _, sub := range subdirs {
			// check if directory
			if fi, err := filepath.EvalSymlinks(sub); err == nil {
				_ = fi
			}
		}
		return nil
	}
	if err := walkFiles(rootDir); err != nil {
		return c, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].absPath < files[j].absPath
	})

	seenDirs := map[string]bool{}
	for _, f := range files {
		dir := filepath.Dir(f.absPath)
		rel, _ := filepath.Rel(rootDir, dir)
		if !seenDirs[rel] {
			seenDirs[rel] = true
		}
		reqs, err := parser.ParseFile(f.absPath)
		if err != nil {
			continue
		}
		c.Files = append(c.Files, model.HTTPFile{Path: f.absPath, Requests: reqs})
	}
	return c, nil
}

func (m *BrowserModel) SetCollection(c *model.Collection) {
	m.collection = c
	m.items = m.buildItems()
}

// buildItems flattens the collection into a displayable list.
func (m BrowserModel) buildItems() []BrowserItem {
	if m.collection == nil {
		return nil
	}

	// Group files by their directory relative to root
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
			// files in root, no dir entry
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
			// directory entry
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
			line = indent + colored + urlPart
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
