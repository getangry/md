package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddToTree(t *testing.T) {
	root := &FileNode{
		Name:  "root",
		Path:  "/tmp/test",
		IsDir: true,
	}

	// Test adding a simple file
	addToTree(root, "/tmp/test", "/tmp/test/file.md", false)

	if len(root.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(root.Children))
	}

	child := root.Children[0]
	if child.Name != "file.md" || child.IsDir {
		t.Errorf("Expected file.md (file), got %s (dir: %t)", child.Name, child.IsDir)
	}

	// Test adding a nested file
	addToTree(root, "/tmp/test", "/tmp/test/subdir/nested.md", false)

	if len(root.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(root.Children))
	}

	// Find the subdir
	var subdir *FileNode
	for _, child := range root.Children {
		if child.Name == "subdir" {
			subdir = child
			break
		}
	}

	if subdir == nil || !subdir.IsDir {
		t.Error("Expected to find subdir directory")
	}

	if len(subdir.Children) != 1 || subdir.Children[0].Name != "nested.md" {
		t.Error("Expected nested.md in subdir")
	}
}

func TestSortTree(t *testing.T) {
	root := &FileNode{
		Name:  "root",
		IsDir: true,
		Children: []*FileNode{
			{Name: "zebra.md", IsDir: false},
			{Name: "dir2", IsDir: true},
			{Name: "alpha.md", IsDir: false},
			{Name: "dir1", IsDir: true},
		},
	}

	sortTree(root)

	// Directories should come first, then files, all alphabetically
	expected := []string{"dir1", "dir2", "alpha.md", "zebra.md"}

	if len(root.Children) != len(expected) {
		t.Errorf("Expected %d children, got %d", len(expected), len(root.Children))
	}

	for i, child := range root.Children {
		if child.Name != expected[i] {
			t.Errorf("Expected child %d to be %s, got %s", i, expected[i], child.Name)
		}
	}

	// Verify directories are marked correctly
	if !root.Children[0].IsDir || !root.Children[1].IsDir {
		t.Error("First two children should be directories")
	}
	if root.Children[2].IsDir || root.Children[3].IsDir {
		t.Error("Last two children should be files")
	}
}

func TestFlattenTree(t *testing.T) {
	root := &FileNode{
		Name:  "root",
		IsDir: true,
		Children: []*FileNode{
			{
				Name:  "dir1",
				IsDir: true,
				Children: []*FileNode{
					{Name: "file1.md", IsDir: false, Path: "/test/dir1/file1.md"},
				},
			},
			{Name: "file2.md", IsDir: false, Path: "/test/file2.md"},
		},
	}

	lines := FlattenTree(root, "", false)

	if len(lines) == 0 {
		t.Error("Expected flattened tree to have lines")
	}

	// Check for proper tree structure indicators
	hasDir := false
	hasFile := false
	for _, line := range lines {
		if strings.Contains(line, "[+]") && strings.HasSuffix(line, "/") {
			hasDir = true
		}
		if strings.Contains(line, "[-]") && strings.HasSuffix(line, ".md") {
			hasFile = true
		}
	}

	if !hasDir || !hasFile {
		t.Error("Expected both directories and files in flattened output")
	}
}

func TestCollectFiles(t *testing.T) {
	root := &FileNode{
		Name:  "root",
		IsDir: true,
		Children: []*FileNode{
			{
				Name:  "dir1",
				IsDir: true,
				Children: []*FileNode{
					{Name: "file1.md", IsDir: false, Path: "/test/dir1/file1.md"},
					{Name: "file2.md", IsDir: false, Path: "/test/dir1/file2.md"},
				},
			},
			{Name: "file3.md", IsDir: false, Path: "/test/file3.md"},
		},
	}

	files := CollectFiles(root)

	expectedFiles := []string{
		"/test/dir1/file1.md",
		"/test/dir1/file2.md",
		"/test/file3.md",
	}

	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(files))
	}

	for i, file := range files {
		if file != expectedFiles[i] {
			t.Errorf("Expected file %d to be %s, got %s", i, expectedFiles[i], file)
		}
	}
}

func TestFindMarkdownFilesQuick(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "md_test_quick")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files - only in root, no subdirs for quick test
	testFiles := []string{
		"README.md",
		"guide.md",
		"code.go", // Should be ignored
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create a subdirectory with markdown - should be ignored by quick scan
	subDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.md"), []byte("nested"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// Test quick finder
	tree, err := FindMarkdownFilesQuick(tempDir, false)
	if err != nil {
		t.Fatalf("FindMarkdownFilesQuick failed: %v", err)
	}

	files := CollectFiles(tree)

	// Should find 2 markdown files in root only (no subdirs)
	if len(files) != 2 {
		t.Errorf("Expected 2 markdown files, got %d: %v", len(files), files)
	}

	// Should include the docs directory (but not scan inside it)
	hasDocsDir := false
	for _, child := range tree.Children {
		if child.Name == "docs" && child.IsDir {
			hasDocsDir = true
			// Should have no children since quick scan doesn't recurse
			if len(child.Children) != 0 {
				t.Error("Quick scan should not recurse into subdirectories")
			}
			break
		}
	}

	if !hasDocsDir {
		t.Error("Quick scan should include directories in tree structure")
	}
}

// Integration test that creates temporary files
func TestFindMarkdownFilesIntegration(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "md_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{
		"README.md",
		"docs/guide.md",
		"src/code.go", // Should be ignored
		".hidden.md",  // Should be ignored
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Test finding markdown files
	tree, err := FindMarkdownFiles(tempDir, false)
	if err != nil {
		t.Fatalf("FindMarkdownFiles failed: %v", err)
	}

	files := CollectFiles(tree)

	// Should find markdown files, excluding .go files
	// Note: .hidden.md might still be found depending on gitignore parsing
	if len(files) < 2 {
		t.Errorf("Expected at least 2 markdown files, got %d: %v", len(files), files)
	}

	// Check that we found the right files
	foundReadme := false
	foundGuide := false
	for _, file := range files {
		if filepath.Base(file) == "README.md" {
			foundReadme = true
		}
		if filepath.Base(file) == "guide.md" {
			foundGuide = true
		}
	}

	if !foundReadme {
		t.Error("Expected to find README.md")
	}
	if !foundGuide {
		t.Error("Expected to find guide.md")
	}
}
