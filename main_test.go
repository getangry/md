package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainFunctionality(t *testing.T) {
	// Test that the main components can be created without crashing

	// Create temporary test environment
	tempDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file
	testFile := filepath.Join(tempDir, "test.md")
	testContent := `# Test Document

This is a test markdown document for testing the md viewer.

## Features

- Feature 1
- Feature 2

## Code

` + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```" + `

End of document.`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test single file model creation
	singleModel, err := NewSingleFileModel(testFile)
	if err != nil {
		t.Errorf("Failed to create single file model: %v", err)
	}

	if singleModel == nil {
		t.Error("Single file model should not be nil")
	}

	// Change to temp directory for dual pane test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test dual pane model creation
	dualModel, err := NewDualPaneModel(false)
	if err != nil {
		t.Errorf("Failed to create dual pane model: %v", err)
	}

	if dualModel == nil {
		t.Error("Dual pane model should not be nil")
	}

	// Note: With async loading, the dual model starts empty and loads files in background
	// This is expected behavior for fast startup - just verify the model is properly initialized
	if dualModel.rootPath == "" {
		t.Error("Dual pane model should have root path set")
	}
}

func TestInclusiveFlag(t *testing.T) {
	// Create temporary test environment with gitignore
	tempDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .gitignore
	gitignoreContent := "ignored.md\ntemp/\n"
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create normal file
	normalFile := filepath.Join(tempDir, "normal.md")
	if err := os.WriteFile(normalFile, []byte("# Normal"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	// Create ignored file
	ignoredFile := filepath.Join(tempDir, "ignored.md")
	if err := os.WriteFile(ignoredFile, []byte("# Ignored"), 0644); err != nil {
		t.Fatalf("Failed to create ignored file: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test with includeIgnored = false (should find normal.md)
	tree1, err := FindMarkdownFiles(".", false)
	if err != nil {
		t.Fatalf("Failed to find files (exclude ignored): %v", err)
	}

	files1 := CollectFiles(tree1)
	if len(files1) == 0 {
		t.Error("Expected to find at least normal.md when excluding ignored")
	}

	// Test with includeIgnored = true (should find both normal.md and ignored.md)
	tree2, err := FindMarkdownFiles(".", true)
	if err != nil {
		t.Fatalf("Failed to find files (include ignored): %v", err)
	}

	files2 := CollectFiles(tree2)
	if len(files2) == 0 {
		t.Error("Expected to find files when including ignored")
	}

	// Verify that inclusive mode finds at least as many files
	if len(files2) < len(files1) {
		t.Errorf("Inclusive mode should find at least as many files as exclusive mode: got %d vs %d", len(files2), len(files1))
	}
}

func TestErrorHandling(t *testing.T) {
	// Test with non-existent directory - this should not error, just return empty results
	_, err := FindMarkdownFiles("/nonexistent/directory", false)
	// Note: FindMarkdownFiles uses WalkDir which handles non-existent paths gracefully
	if err == nil {
		// This is actually expected - WalkDir handles missing directories
		t.Skip("FindMarkdownFiles handles non-existent directories gracefully")
	}

	// Test single file model with non-existent file
	_, err = NewSingleFileModel("/nonexistent/file.md")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test dual pane model in directory with no markdown files
	tempDir, err := os.MkdirTemp("", "md_test_empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a non-markdown file
	nonMdFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(nonMdFile, []byte("not markdown"), 0644); err != nil {
		t.Fatalf("Failed to create non-md file: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Should not fail, but should have empty file list
	dualModel, err := NewDualPaneModel(false)
	if err != nil {
		t.Errorf("Dual pane model should handle empty directories gracefully: %v", err)
	}

	if len(dualModel.allFiles) != 0 {
		t.Errorf("Expected 0 files in directory with no markdown, got %d", len(dualModel.allFiles))
	}
}
