package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

var (
	rendererCache = make(map[int]*glamour.TermRenderer)
	rendererMutex sync.RWMutex
)

type SingleFileModel struct {
	filepath        string
	content         string
	lines           []string
	viewport        int
	width           int
	height          int
	renderer        *glamour.TermRenderer
	raw             bool // Toggle between raw and rendered view
	contentLoaded   bool // Track if content has been loaded
	rendererCreated bool // Track if renderer has been created
}

func NewSingleFileModel(filepath string) (*SingleFileModel, error) {
	// No file system operations here - completely instant startup
	m := &SingleFileModel{
		filepath:        filepath,
		content:         "", // Will be loaded lazily
		viewport:        0,
		renderer:        nil,                         // Will be created lazily when needed
		raw:             false,                       // Default to rendered mode
		lines:           []string{"Loading file..."}, // Placeholder
		contentLoaded:   false,
		rendererCreated: false,
	}

	return m, nil
}

func NewSingleFileModelWithContent(filepath string, content string) (*SingleFileModel, error) {
	// For stdin or pre-loaded content - instant startup with content
	m := &SingleFileModel{
		filepath:        filepath,
		content:         content,
		viewport:        0,
		renderer:        nil,                    // Will be created lazily when needed
		raw:             false,                  // Default to rendered mode
		lines:           []string{"Loading..."}, // Will be replaced immediately
		contentLoaded:   true,                   // Content is already available
		rendererCreated: false,                  // Renderer still needs to be created
	}

	return m, nil
}

func (m *SingleFileModel) Init() tea.Cmd {
	// If content is already loaded (stdin), don't load from file
	if m.contentLoaded {
		return func() tea.Msg {
			return fileLoadedMsg{content: m.content, err: nil}
		}
	}

	// Load file content in true background goroutine
	return tea.Tick(1, func(t time.Time) tea.Msg {
		// This runs in a separate goroutine, not blocking UI
		content, err := os.ReadFile(m.filepath)
		if err != nil {
			return fileLoadedMsg{content: "", err: err}
		}
		return fileLoadedMsg{content: string(content), err: nil}
	})
}

type fileLoadedMsg struct {
	content string
	err     error
}

type renderContentMsg struct{}

type rendererCreatedMsg struct {
	renderer *glamour.TermRenderer
	err      error
}

type contentRenderedMsg struct {
	lines []string
	err   error
}

func createRendererInBackground(width int) tea.Cmd {
	return tea.Tick(1, func(t time.Time) tea.Msg {
		// Check cache first
		rendererMutex.RLock()
		if cached, exists := rendererCache[width]; exists {
			rendererMutex.RUnlock()
			return rendererCreatedMsg{renderer: cached, err: nil}
		}
		rendererMutex.RUnlock()

		// Create renderer with fast dark style
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)

		// Cache successful renderer
		if err == nil {
			rendererMutex.Lock()
			rendererCache[width] = renderer
			rendererMutex.Unlock()
		}

		return rendererCreatedMsg{renderer: renderer, err: err}
	})
}

func renderContentAsync(content string, renderer *glamour.TermRenderer, raw bool) tea.Cmd {
	return tea.Tick(1, func(t time.Time) tea.Msg {
		if raw || renderer == nil || content == "" {
			return contentRenderedMsg{lines: strings.Split(content, "\n"), err: nil}
		}

		rendered, err := renderer.Render(content)
		if err != nil {
			return contentRenderedMsg{lines: strings.Split(content, "\n"), err: err}
		}

		return contentRenderedMsg{lines: strings.Split(rendered, "\n"), err: nil}
	})
}

func (m *SingleFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fileLoadedMsg:
		if msg.err != nil {
			m.lines = []string{fmt.Sprintf("Error loading file: %v", msg.err)}
			return m, nil
		}
		m.content = msg.content
		m.contentLoaded = true

		// Show raw content immediately for instant display
		m.lines = strings.Split(m.content, "\n")

		// Start async renderer creation if needed
		if !m.raw && m.renderer == nil {
			width := 80
			if m.width > 0 {
				width = m.width
			}
			return m, createRendererInBackground(width)
		}

		// If we already have a renderer, start async rendering
		if !m.raw && m.renderer != nil {
			return m, renderContentAsync(m.content, m.renderer, m.raw)
		}

		return m, nil

	case rendererCreatedMsg:
		if msg.err != nil {
			// Renderer creation failed - stay in raw mode
			return m, nil
		}

		m.renderer = msg.renderer

		// Now that renderer is ready, start content rendering
		if !m.raw && m.content != "" {
			return m, renderContentAsync(m.content, m.renderer, m.raw)
		}
		return m, nil

	case contentRenderedMsg:
		if msg.err == nil {
			// Successfully rendered
			m.lines = msg.lines
		} else {
			// If rendering failed, fall back to raw content
			m.lines = strings.Split(m.content, "\n")
		}
		return m, nil

	case renderContentMsg:
		// Manual refresh trigger
		if m.content != "" && m.renderer != nil {
			return m, renderContentAsync(m.content, m.renderer, m.raw)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height // Use full height for content

		// Re-create renderer with new width
		if !m.raw && m.width > 0 && m.content != "" {
			m.renderer = nil // Force recreation with new width
			return m, createRendererInBackground(m.width)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "j", "down":
			if m.viewport < len(m.lines)-m.height {
				m.viewport++
			}

		case "k", "up":
			if m.viewport > 0 {
				m.viewport--
			}

		case "ctrl+d", "pgdown":
			m.viewport += m.height / 2
			if m.viewport > len(m.lines)-m.height {
				m.viewport = max(0, len(m.lines)-m.height)
			}

		case "ctrl+u", "pgup":
			m.viewport -= m.height / 2
			if m.viewport < 0 {
				m.viewport = 0
			}

		case "g", "home":
			m.viewport = 0

		case "G", "end":
			m.viewport = max(0, len(m.lines)-m.height)

		case "r":
			// Toggle raw/rendered view
			m.raw = !m.raw
			if m.content != "" && m.renderer != nil {
				return m, renderContentAsync(m.content, m.renderer, m.raw)
			}

		case " ":
			// Space for page down
			m.viewport += m.height - 1
			if m.viewport > len(m.lines)-m.height {
				m.viewport = max(0, len(m.lines)-m.height)
			}
		}
	}

	return m, nil
}

func (m *SingleFileModel) View() string {
	if m.height == 0 {
		return "Loading..."
	}

	// Simple content view without heavy status bar
	var content strings.Builder
	endLine := min(m.viewport+m.height, len(m.lines))

	for i := m.viewport; i < endLine; i++ {
		content.WriteString(m.lines[i])
		if i < endLine-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}
