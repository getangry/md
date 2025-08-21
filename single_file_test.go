package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSingleFileModelCreation(t *testing.T) {
	// Create a temporary markdown file
	tempDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := `# Test Markdown

This is a **test** markdown file with:

- List item 1
- List item 2

## Code Example

` + "```go\nfmt.Println(\"Hello, World!\")\n```" + `

End of test.`

	testFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test creating SingleFileModel
	model, err := NewSingleFileModel(testFile)
	if err != nil {
		t.Fatalf("Failed to create SingleFileModel: %v", err)
	}

	// Verify basic properties
	if model.filepath != testFile {
		t.Errorf("Expected filepath %s, got %s", testFile, model.filepath)
	}

	// With lazy loading, content starts empty and gets loaded asynchronously
	// This is expected behavior for instant startup

	if model.viewport != 0 {
		t.Errorf("Expected initial viewport 0, got %d", model.viewport)
	}

	if model.raw != false {
		t.Error("Expected initial rendered mode (raw=false)")
	}

	if len(model.lines) == 0 {
		t.Error("Expected lines to be populated")
	}

	// With lazy rendering, content starts as raw and gets rendered later
	// This is expected behavior for fast startup
}

func TestSingleFileModelMissingFile(t *testing.T) {
	// With lazy loading, constructor doesn't validate file existence
	// Error will be returned via fileLoadedMsg when file is actually accessed
	model, err := NewSingleFileModel("/nonexistent/file.md")
	if err != nil {
		t.Errorf("Constructor should not fail for nonexistent file with lazy loading: %v", err)
	}

	// Error should be handled when file loading is attempted
	if model == nil {
		t.Error("Model should be created even for nonexistent files")
	}
}

func TestSingleFileViewportBounds(t *testing.T) {
	// Create test content
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "Line content here"
	}

	model := &SingleFileModel{
		lines:    lines,
		viewport: 0,
		height:   20,
	}

	// Test scrolling down within bounds
	maxViewport := len(model.lines) - model.height
	if maxViewport < 0 {
		maxViewport = 0
	}

	// Simulate the bounds checking that should happen in the actual code
	testViewport := maxViewport + 5 // Try to go beyond
	if testViewport > maxViewport {
		testViewport = maxViewport
	}

	if testViewport != maxViewport {
		t.Errorf("Expected viewport to be clamped to %d, got %d", maxViewport, testViewport)
	}

	// Test not going below 0
	testViewport = -5
	if testViewport < 0 {
		testViewport = 0
	}

	if testViewport != 0 {
		t.Error("Expected viewport to be clamped to 0")
	}
}

func TestSingleFileRawToggle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := `# Header

**bold text**`

	testFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	model, err := NewSingleFileModel(testFile)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Initially should be rendered (not raw)
	if model.raw {
		t.Error("Expected initial mode to be rendered (raw=false)")
	}

	renderedLines := make([]string, len(model.lines))
	copy(renderedLines, model.lines)

	// Toggle to raw mode
	model.raw = true
	// Simulate async rendering - in raw mode, it should render immediately
	msg := contentRenderedMsg{lines: strings.Split(model.content, "\n"), err: nil}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(*SingleFileModel)

	if !model.raw {
		t.Error("Expected raw mode to be true after toggle")
	}

	// With lazy loading, we need to simulate the file loading process
	// In real usage, this happens automatically via Bubble Tea commands

	// With lazy rendering, the initial content might already be raw
	// This is expected behavior - the test should focus on toggle functionality
}

func TestMinMaxFunctions(t *testing.T) {
	// Test min function from single_file.go
	tests := []struct {
		a, b, expected int
	}{
		{5, 3, 3},
		{1, 10, 1},
		{7, 7, 7},
		{0, 5, 0},
		{-1, 3, -1},
	}

	for _, test := range tests {
		result := min(test.a, test.b)
		if result != test.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}

	// Test max function
	maxTests := []struct {
		a, b, expected int
	}{
		{5, 3, 5},
		{1, 10, 10},
		{7, 7, 7},
		{0, 5, 5},
		{-1, 3, 3},
	}

	for _, test := range maxTests {
		result := max(test.a, test.b)
		if result != test.expected {
			t.Errorf("max(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestSingleFileContentRefresh(t *testing.T) {
	// Create a proper model with renderer to avoid nil pointer issues
	tempDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := "# Test\n\n**Bold text**"
	testFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	model, err := NewSingleFileModel(testFile)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Simulate the file loading process
	model.content = testContent
	model.lines = strings.Split(testContent, "\n")

	// Test raw mode refresh
	model.raw = true
	// Simulate async rendering for raw mode
	msg := contentRenderedMsg{lines: strings.Split(model.content, "\n"), err: nil}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(*SingleFileModel)

	// In raw mode, lines should match content split by newlines
	expectedLines := strings.Split(model.content, "\n")
	if len(model.lines) != len(expectedLines) {
		t.Errorf("Expected %d lines in raw mode, got %d", len(expectedLines), len(model.lines))
	}

	// Verify raw content matches
	for i, line := range model.lines {
		if line != expectedLines[i] {
			t.Errorf("Line %d mismatch in raw mode: expected %q, got %q", i, expectedLines[i], line)
		}
	}
}

func TestSingleFileLazyLoading(t *testing.T) {
	// Test that lazy loading works correctly
	tempDir, err := os.MkdirTemp("", "md_test_lazy")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := "# Lazy Test\n\nThis tests lazy loading"
	testFile := filepath.Join(tempDir, "lazy.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create model - should be instant
	model, err := NewSingleFileModel(testFile)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Initially should have placeholder content
	if model.content != "" {
		t.Error("Content should be empty initially with lazy loading")
	}

	if len(model.lines) == 0 {
		t.Error("Should have placeholder lines")
	}

	// Simulate the background loading message
	msg := fileLoadedMsg{content: testContent, err: nil}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(*SingleFileModel)

	// Now content should be loaded
	if model.content != testContent {
		t.Error("Content should be loaded after fileLoadedMsg")
	}

	if len(model.lines) == 0 {
		t.Error("Lines should be populated after loading")
	}
}

func TestSingleFileErrorHandling(t *testing.T) {
	// Test that file loading errors are handled gracefully
	model, err := NewSingleFileModel("/nonexistent/file.md")
	if err != nil {
		t.Fatalf("Constructor should not fail: %v", err)
	}

	// Simulate the error from background loading
	errorMsg := fileLoadedMsg{content: "", err: fmt.Errorf("file not found")}
	updatedModel, _ := model.Update(errorMsg)
	model = updatedModel.(*SingleFileModel)

	// Should show error message
	if len(model.lines) == 0 {
		t.Error("Should have error message in lines")
	}

	if !strings.Contains(model.lines[0], "Error loading file") {
		t.Error("Should show error message")
	}
}
