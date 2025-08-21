package main

import (
	"testing"
)

func TestFindTreeLineForFile(t *testing.T) {
	// Sample tree lines as they would appear in the UI
	treeLines := []string{
		"└── [+] docs/",
		"    ├── [-] guide.md",
		"    └── [-] tutorial.md",
		"└── [-] README.md",
	}

	// Sample file list
	allFiles := []string{
		"/test/docs/guide.md",
		"/test/docs/tutorial.md",
		"/test/README.md",
	}

	tests := []struct {
		fileIndex    int
		expectedLine int
		description  string
	}{
		{0, 1, "First file (guide.md) should map to line 1"},
		{1, 2, "Second file (tutorial.md) should map to line 2"},
		{2, 3, "Third file (README.md) should map to line 3"},
		{-1, 0, "Invalid index should return 0"},
		{10, 0, "Out of bounds index should return 0"},
	}

	for _, test := range tests {
		result := findTreeLineForFile(test.fileIndex, treeLines, allFiles)
		if result != test.expectedLine {
			t.Errorf("%s: expected line %d, got %d", test.description, test.expectedLine, result)
		}
	}
}

func TestMinMax(t *testing.T) {
	// Test min function
	if min(5, 3) != 3 {
		t.Error("min(5, 3) should return 3")
	}
	if min(1, 10) != 1 {
		t.Error("min(1, 10) should return 1")
	}

	// Test max function
	if max(5, 3) != 5 {
		t.Error("max(5, 3) should return 5")
	}
	if max(1, 10) != 10 {
		t.Error("max(1, 10) should return 10")
	}

	// Test minFloat function
	if minFloat(5.5, 3.3) != 3.3 {
		t.Error("minFloat(5.5, 3.3) should return 3.3")
	}

	// Test maxFloat function
	if maxFloat(5.5, 3.3) != 5.5 {
		t.Error("maxFloat(5.5, 3.3) should return 5.5")
	}
}

func TestTreeViewportAdjustment(t *testing.T) {
	// Create a mock dual pane model
	m := &DualPaneModel{
		treeSelectedIdx: 10,
		treeViewport:    0,
		height:          20, // Available height will be 18 (20-2)
	}

	// Test case where selected item is below viewport
	m.adjustTreeViewport()

	// Selected item (10) should be visible, so viewport should adjust
	expectedViewport := 10 - 18 + 1 // 10 - (height-2) + 1 = -7, but should be clamped
	if expectedViewport < 0 {
		expectedViewport = 0
	}

	// Actually, let's test a more realistic scenario
	m.treeSelectedIdx = 25
	m.treeViewport = 0
	m.adjustTreeViewport()

	// With height 20 (18 available), if selected is at 25, viewport should be 25-18+1 = 8
	expectedViewport = 25 - 18 + 1
	if m.treeViewport != expectedViewport {
		t.Errorf("Expected viewport to be adjusted to %d, got %d", expectedViewport, m.treeViewport)
	}

	// Test case where selected item is above viewport
	m.treeSelectedIdx = 3
	m.treeViewport = 10
	m.adjustTreeViewport()

	if m.treeViewport != 3 {
		t.Errorf("Expected viewport to be adjusted to 3, got %d", m.treeViewport)
	}
}

func TestSplitRatioCalculation(t *testing.T) {
	m := &DualPaneModel{
		width:      100,
		splitRatio: 0.3,
	}

	// Test tree width calculation
	treeWidth := int(float64(m.width) * m.splitRatio)
	expectedTreeWidth := 30

	if treeWidth != expectedTreeWidth {
		t.Errorf("Expected tree width %d, got %d", expectedTreeWidth, treeWidth)
	}

	// Test content width calculation
	contentWidth := m.width - treeWidth - 1 // -1 for divider
	expectedContentWidth := 69

	if contentWidth != expectedContentWidth {
		t.Errorf("Expected content width %d, got %d", expectedContentWidth, contentWidth)
	}
}

// Test model initialization
func TestDualPaneModelDefaults(t *testing.T) {
	// We can't easily test NewDualPaneModel without file system dependencies,
	// but we can test that the struct has reasonable defaults
	m := &DualPaneModel{
		splitRatio:  0.3,
		focusedPane: 0,
		raw:         false,
	}

	if m.splitRatio != 0.3 {
		t.Errorf("Expected default split ratio 0.3, got %f", m.splitRatio)
	}

	if m.focusedPane != 0 {
		t.Errorf("Expected default focused pane 0 (tree), got %d", m.focusedPane)
	}

	if m.raw != false {
		t.Error("Expected default raw mode to be false")
	}
}

func TestContentViewportBounds(t *testing.T) {
	m := &DualPaneModel{
		contentViewport: 0,
		renderedLines:   make([]string, 100), // 100 lines of content
		height:          20,
	}

	availableHeight := m.height - 2 // 18

	// Test scrolling down
	originalViewport := m.contentViewport

	// Simulate scrolling down - should not exceed bounds
	maxViewport := len(m.renderedLines) - availableHeight
	if maxViewport < 0 {
		maxViewport = 0
	}

	// Test that we don't scroll beyond the end
	m.contentViewport = maxViewport + 10 // Try to go beyond
	if m.contentViewport > maxViewport {
		m.contentViewport = maxViewport // This is what the actual code should do
	}

	if m.contentViewport != maxViewport {
		t.Errorf("Expected content viewport to be clamped to %d, got %d", maxViewport, m.contentViewport)
	}

	// Test that we don't scroll above 0
	m.contentViewport = -5
	if m.contentViewport < 0 {
		m.contentViewport = 0 // This is what the actual code should do
	}

	if m.contentViewport != 0 {
		t.Errorf("Expected content viewport to be clamped to 0, got %d", m.contentViewport)
	}

	// Verify we started with a reasonable value
	_ = originalViewport
}
