package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"sync"
)

var (
	dualRendererCache = make(map[int]*glamour.TermRenderer)
	dualRendererMutex sync.RWMutex
)

type DualPaneModel struct {
	fileTree        *FileNode
	allFiles        []string
	treeLines       []string
	selectedIndex   int
	treeViewport    int
	contentViewport int
	currentContent  string
	renderedLines   []string
	width           int
	height          int
	splitRatio      float64
	renderer        *glamour.TermRenderer
	focusedPane     int // 0 = tree, 1 = content
	raw             bool
	treeSelectedIdx int // Index of selected line in treeLines
	includeIgnored  bool
	rootPath        string
	isExpanding     bool // True when background expansion is happening
	currentDepth    int  // Current scan depth
}

func NewDualPaneModel(includeIgnored bool) (*DualPaneModel, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Start with completely empty state for instant startup
	emptyTree := &FileNode{
		Name:  "Loading...",
		Path:  cwd,
		IsDir: true,
	}

	m := &DualPaneModel{
		fileTree:        emptyTree,
		allFiles:        []string{},
		treeLines:       []string{"Loading markdown files..."},
		selectedIndex:   0,
		treeSelectedIdx: 0,
		splitRatio:      0.3,
		renderer:        nil, // Will be created lazily when needed
		focusedPane:     0,
		includeIgnored:  includeIgnored,
		rootPath:        cwd,
		currentDepth:    -1, // -1 indicates not started yet
		isExpanding:     false,
	}

	return m, nil
}

func (m *DualPaneModel) Init() tea.Cmd {
	// Start initial scan immediately
	return func() tea.Msg {
		return initialLoadMsg{}
	}
}

type initialLoadMsg struct{}

type loadCompleteMsg struct {
	tree *FileNode
	err  error
}

type expandTreeMsg struct{}

func (m *DualPaneModel) expandTree() tea.Cmd {
	if m.isExpanding {
		return nil
	}

	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return performExpansionMsg{}
	})
}

type performExpansionMsg struct{}

func (m *DualPaneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initialLoadMsg:
		// Perform initial load of depth 0 files
		if m.currentDepth == -1 {
			m.currentDepth = 0
			m.isExpanding = true

			// Run in background to avoid blocking UI
			return m, func() tea.Msg {
				fileTree, err := FindMarkdownFilesQuick(m.rootPath, m.includeIgnored)
				if err != nil {
					return loadCompleteMsg{err: err}
				}
				return loadCompleteMsg{tree: fileTree, err: nil}
			}
		}
		return m, nil

	case loadCompleteMsg:
		if msg.err != nil {
			m.treeLines = []string{"Error loading files: " + msg.err.Error()}
			m.isExpanding = false
			return m, nil
		}

		m.fileTree = msg.tree
		m.allFiles = CollectFiles(msg.tree)
		m.treeLines = FlattenTree(msg.tree, "", false)
		m.isExpanding = false

		// Load first file if available
		if len(m.allFiles) > 0 {
			m.selectedIndex = 0
			m.treeSelectedIdx = findTreeLineForFile(0, m.treeLines, m.allFiles)
			m.loadFile(0)
		}

		// Start background expansion to deeper levels
		return m, tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
			return expandTreeMsg{}
		})

	case expandTreeMsg:
		return m, m.expandTree()

	case performExpansionMsg:
		if !m.isExpanding {
			m.isExpanding = true
			// Expand to next depth level
			m.currentDepth++
			newTree, err := FindMarkdownFilesWithDepth(m.rootPath, m.includeIgnored, m.currentDepth)
			if err == nil {
				newFiles := CollectFiles(newTree)
				if len(newFiles) > len(m.allFiles) {
					// We found new files, update the model
					m.fileTree = newTree
					m.allFiles = newFiles
					m.treeLines = FlattenTree(newTree, "", false)

					// Preserve selection if possible
					if m.selectedIndex < len(m.allFiles) {
						m.treeSelectedIdx = findTreeLineForFile(m.selectedIndex, m.treeLines, m.allFiles)
					}
				}
			}
			m.isExpanding = false

			// Continue expanding if we haven't hit a reasonable depth limit
			if m.currentDepth < 5 {
				return m, m.expandTree()
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height - 2 // Reserve space for status bar

		// Update renderer width based on content pane width
		m.updateRendererWidth()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			// Switch focus between panes
			m.focusedPane = (m.focusedPane + 1) % 2

		case "h", "left":
			if m.focusedPane == 1 {
				m.focusedPane = 0
			}

		case "l", "right":
			if m.focusedPane == 0 {
				m.focusedPane = 1
			}

		case "j", "down":
			if m.focusedPane == 0 {
				// Tree navigation
				if m.selectedIndex < len(m.allFiles)-1 {
					m.selectedIndex++
					m.treeSelectedIdx = findTreeLineForFile(m.selectedIndex, m.treeLines, m.allFiles)
					m.loadFile(m.selectedIndex)
					m.adjustTreeViewport()
				}
			} else {
				// Content scrolling
				availableHeight := m.height - 2
				if m.contentViewport < len(m.renderedLines)-availableHeight {
					m.contentViewport++
				}
			}

		case "k", "up":
			if m.focusedPane == 0 {
				// Tree navigation
				if m.selectedIndex > 0 {
					m.selectedIndex--
					m.treeSelectedIdx = findTreeLineForFile(m.selectedIndex, m.treeLines, m.allFiles)
					m.loadFile(m.selectedIndex)
					m.adjustTreeViewport()
				}
			} else {
				// Content scrolling
				if m.contentViewport > 0 {
					m.contentViewport--
				}
			}

		case "enter":
			if m.focusedPane == 0 && m.selectedIndex >= 0 && m.selectedIndex < len(m.allFiles) {
				m.focusedPane = 1
				m.contentViewport = 0
			}

		case "ctrl+d", "pgdown":
			if m.focusedPane == 1 {
				availableHeight := m.height - 2
				m.contentViewport += availableHeight / 2
				if m.contentViewport > len(m.renderedLines)-availableHeight {
					m.contentViewport = max(0, len(m.renderedLines)-availableHeight)
				}
			}

		case "ctrl+u", "pgup":
			if m.focusedPane == 1 {
				availableHeight := m.height - 2
				m.contentViewport -= availableHeight / 2
				if m.contentViewport < 0 {
					m.contentViewport = 0
				}
			}

		case "g", "home":
			if m.focusedPane == 1 {
				m.contentViewport = 0
			} else {
				m.selectedIndex = 0
				m.treeSelectedIdx = findTreeLineForFile(0, m.treeLines, m.allFiles)
				m.treeViewport = 0
				if len(m.allFiles) > 0 {
					m.loadFile(0)
				}
			}

		case "G", "end":
			if m.focusedPane == 1 {
				availableHeight := m.height - 2
				m.contentViewport = max(0, len(m.renderedLines)-availableHeight)
			} else {
				m.selectedIndex = len(m.allFiles) - 1
				m.treeSelectedIdx = findTreeLineForFile(m.selectedIndex, m.treeLines, m.allFiles)
				m.loadFile(m.selectedIndex)
				m.adjustTreeViewport()
			}

		case "r":
			// Toggle raw/rendered view
			m.raw = !m.raw
			m.refreshContent()

		case "<", "{":
			// Decrease split ratio
			m.splitRatio = maxFloat(0.2, m.splitRatio-0.05)
			m.updateRendererWidth()

		case ">", "}":
			// Increase split ratio
			m.splitRatio = minFloat(0.5, m.splitRatio+0.05)
			m.updateRendererWidth()

		case "e":
			// Manual expand - scan deeper
			if !m.isExpanding {
				return m, m.expandTree()
			}
		}

	case tea.MouseMsg:
		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			// Calculate which pane the mouse is in based on x coordinate
			treeWidth := int(float64(m.width) * m.splitRatio)

			if msg.X < treeWidth {
				// Mouse is in tree pane - scroll tree
				if msg.Type == tea.MouseWheelUp && m.treeViewport > 0 {
					m.treeViewport--
				} else if msg.Type == tea.MouseWheelDown && m.treeViewport < len(m.treeLines)-(m.height-2) {
					m.treeViewport++
				}
			} else {
				// Mouse is in content pane - scroll content
				availableHeight := m.height - 2
				if msg.Type == tea.MouseWheelUp && m.contentViewport > 0 {
					m.contentViewport--
				} else if msg.Type == tea.MouseWheelDown && m.contentViewport < len(m.renderedLines)-availableHeight {
					m.contentViewport++
				}
			}
		}
	}

	return m, nil
}

func (m *DualPaneModel) View() string {
	if m.height == 0 {
		return "Loading..."
	}

	treeWidth := int(float64(m.width) * m.splitRatio)
	contentWidth := m.width - treeWidth - 1 - 1 // -1 for divider, -1 for scroll bar

	// Styles
	focusedStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	unfocusedStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("250")).
		Width(m.width).
		Padding(0, 1)

	// Calculate available content height (subtract border padding)
	availableHeight := m.height - 2 // -2 for top and bottom borders

	// Build tree view
	var treeContent strings.Builder
	for i := 0; i < availableHeight; i++ {
		lineIdx := m.treeViewport + i
		if lineIdx < len(m.treeLines) {
			line := m.treeLines[lineIdx]

			// Check if this is the selected line
			isSelected := (lineIdx == m.treeSelectedIdx)

			// Add cursor prefix for selected line
			var displayLine string
			if isSelected && m.focusedPane == 0 {
				displayLine = "\u276f " + line
			} else {
				displayLine = "  " + line
			}

			// Truncate line to fit width using proper character width
			maxWidth := treeWidth - 4 // Account for border padding
			if maxWidth > 0 && runewidth.StringWidth(displayLine) > maxWidth {
				if maxWidth > 3 {
					displayLine = runewidth.Truncate(displayLine, maxWidth-3, "...")
				} else {
					displayLine = runewidth.Truncate(displayLine, maxWidth, "")
				}
			}

			// Apply background highlight for selected item
			if isSelected && m.focusedPane == 0 {
				displayLine = selectedStyle.Render(displayLine)
			}

			treeContent.WriteString(displayLine)
		}
		if i < availableHeight-1 {
			treeContent.WriteString("\n")
		}
	}

	// Build content view
	var contentView strings.Builder
	endLine := min(m.contentViewport+availableHeight, len(m.renderedLines))

	for i := m.contentViewport; i < endLine; i++ {
		line := m.renderedLines[i]
		// Don't truncate content lines - let them wrap naturally
		// The renderer should handle word wrapping
		contentView.WriteString(line)
		if i < endLine-1 {
			contentView.WriteString("\n")
		}
	}

	// Fill remaining space
	linesShown := endLine - m.contentViewport
	for i := linesShown; i < availableHeight; i++ {
		contentView.WriteString("\n")
	}

	// Build scroll indicator
	var scrollBar strings.Builder
	if len(m.renderedLines) > 0 && availableHeight > 0 {
		// Calculate scroll position
		totalLines := len(m.renderedLines)
		viewportSize := availableHeight
		scrollTop := m.contentViewport

		// Calculate scroll bar dimensions
		scrollBarHeight := availableHeight
		thumbHeight := max(1, (viewportSize*scrollBarHeight)/totalLines)
		thumbPosition := (scrollTop * (scrollBarHeight - thumbHeight)) / max(1, totalLines-viewportSize)

		for i := 0; i < scrollBarHeight; i++ {
			if i >= thumbPosition && i < thumbPosition+thumbHeight {
				scrollBar.WriteString("█") // Solid block for thumb
			} else {
				scrollBar.WriteString("░") // Light shade for track
			}
			if i < scrollBarHeight-1 {
				scrollBar.WriteString("\n")
			}
		}
	} else {
		// Empty scroll bar
		for i := 0; i < availableHeight; i++ {
			scrollBar.WriteString("░")
			if i < availableHeight-1 {
				scrollBar.WriteString("\n")
			}
		}
	}

	// Combine content with scroll bar
	contentWithScroll := lipgloss.JoinHorizontal(lipgloss.Top,
		contentView.String(),
		scrollBar.String())

	// Apply borders based on focus
	var treePane, contentPane string
	if m.focusedPane == 0 {
		treePane = focusedStyle.
			Width(treeWidth).
			Height(availableHeight).
			Render(treeContent.String())
		contentPane = unfocusedStyle.
			Width(contentWidth + 1). // +1 for scroll bar
			Height(availableHeight).
			Render(contentWithScroll)
	} else {
		treePane = unfocusedStyle.
			Width(treeWidth).
			Height(availableHeight).
			Render(treeContent.String())
		contentPane = focusedStyle.
			Width(contentWidth + 1). // +1 for scroll bar
			Height(availableHeight).
			Render(contentWithScroll)
	}

	// Join panes
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, treePane, contentPane)

	// Create status bar
	currentFile := "No file selected"
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.allFiles) {
		currentFile = m.allFiles[m.selectedIndex]
	}

	viewMode := "Rendered"
	if m.raw {
		viewMode = "Raw"
	}

	focusIndicator := "Tree"
	if m.focusedPane == 1 {
		focusIndicator = "Content"
	}

	// Add expansion indicator
	expansionStatus := ""
	if m.currentDepth == -1 {
		expansionStatus = " | Initializing..."
	} else if m.isExpanding {
		expansionStatus = " | Scanning..."
	} else if m.currentDepth > 0 {
		expansionStatus = fmt.Sprintf(" | Depth %d", m.currentDepth)
	}

	status := fmt.Sprintf("* %s | %s | Focus: %s%s | [tab]switch [e]xpand [q]uit [r]aw/render [<>]resize",
		currentFile,
		viewMode,
		focusIndicator,
		expansionStatus,
	)

	return mainView + "\n" + statusStyle.Render(status)
}

func (m *DualPaneModel) ensureRenderer() {
	if m.renderer == nil {
		width := 60 // Default width

		// Check cache first
		dualRendererMutex.RLock()
		if cached, exists := dualRendererCache[width]; exists {
			m.renderer = cached
			dualRendererMutex.RUnlock()
			return
		}
		dualRendererMutex.RUnlock()

		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			m.renderer = renderer
			// Cache it
			dualRendererMutex.Lock()
			dualRendererCache[width] = renderer
			dualRendererMutex.Unlock()
		}
	}
}

func (m *DualPaneModel) loadFile(index int) {
	if index < 0 || index >= len(m.allFiles) {
		return
	}

	content, err := os.ReadFile(m.allFiles[index])
	if err != nil {
		m.currentContent = fmt.Sprintf("Error loading file: %v", err)
		m.renderedLines = strings.Split(m.currentContent, "\n")
		return
	}

	m.currentContent = string(content)
	m.refreshContent()
	m.contentViewport = 0
}

func (m *DualPaneModel) refreshContent() {
	if m.raw {
		m.renderedLines = strings.Split(m.currentContent, "\n")
	} else {
		m.ensureRenderer()
		if m.renderer != nil {
			rendered, err := m.renderer.Render(m.currentContent)
			if err != nil {
				rendered = m.currentContent
			}
			m.renderedLines = strings.Split(rendered, "\n")
		} else {
			// Fallback to raw if renderer creation failed
			m.renderedLines = strings.Split(m.currentContent, "\n")
		}
	}
}

func (m *DualPaneModel) adjustTreeViewport() {
	availableHeight := m.height - 2
	// Ensure selected tree line is visible
	if m.treeSelectedIdx < m.treeViewport {
		m.treeViewport = m.treeSelectedIdx
	} else if m.treeSelectedIdx >= m.treeViewport+availableHeight {
		m.treeViewport = m.treeSelectedIdx - availableHeight + 1
	}
}

func (m *DualPaneModel) updateRendererWidth() {
	contentWidth := int(float64(m.width) * (1 - m.splitRatio))
	// Account for border padding and ensure minimum width
	wrappingWidth := contentWidth - 6 // -6 for border and padding
	if wrappingWidth < 40 {
		wrappingWidth = 40 // Minimum readable width
	}

	// Check cache first
	dualRendererMutex.RLock()
	if cached, exists := dualRendererCache[wrappingWidth]; exists {
		m.renderer = cached
		dualRendererMutex.RUnlock()
		return
	}
	dualRendererMutex.RUnlock()

	// Create renderer with fast dark style
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(wrappingWidth),
	)

	// Cache successful renderer
	if err == nil {
		m.renderer = renderer
		dualRendererMutex.Lock()
		dualRendererCache[wrappingWidth] = renderer
		dualRendererMutex.Unlock()
	}
	if err == nil {
		m.renderer = renderer
		m.refreshContent()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func findTreeLineForFile(fileIndex int, treeLines []string, allFiles []string) int {
	if fileIndex < 0 || fileIndex >= len(allFiles) {
		return 0
	}

	targetFile := allFiles[fileIndex]
	// Extract just the filename without path
	parts := strings.Split(targetFile, "/")
	filename := parts[len(parts)-1]

	for i, line := range treeLines {
		// Look for lines that contain the filename and are file entries (have [-])
		if strings.Contains(line, "[-]") && strings.Contains(line, filename) {
			return i
		}
	}

	return 0 // Default to first line if not found
}
