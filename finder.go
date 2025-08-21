package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/denormal/go-gitignore"
)

type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*FileNode
}

func FindMarkdownFiles(rootPath string, includeIgnored bool) (*FileNode, error) {
	return FindMarkdownFilesWithDepth(rootPath, includeIgnored, -1)
}

func FindMarkdownFilesQuick(rootPath string, includeIgnored bool) (*FileNode, error) {
	// Ultra-fast scan of just the current directory (no subdirs)
	var ignore gitignore.GitIgnore

	if !includeIgnored {
		gitignorePath := filepath.Join(rootPath, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			ignore, _ = gitignore.NewFromFile(gitignorePath)
		}
	}

	root := &FileNode{
		Name:  filepath.Base(rootPath),
		Path:  rootPath,
		IsDir: true,
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return root, nil // Return empty root on error
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(rootPath, name)

		// Skip hidden files/dirs starting with .
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Check gitignore
		if !includeIgnored && ignore != nil {
			if ignore.Ignore(name) {
				continue
			}
		}

		// Only include markdown files and directories
		if entry.IsDir() {
			// Add directory to tree
			addToTree(root, rootPath, fullPath, true)
		} else if strings.HasSuffix(strings.ToLower(name), ".md") {
			// Add markdown file to tree
			addToTree(root, rootPath, fullPath, false)
		}
	}

	// Sort children
	sortTree(root)
	return root, nil
}

func FindMarkdownFilesWithDepth(rootPath string, includeIgnored bool, maxDepth int) (*FileNode, error) {
	var ignore gitignore.GitIgnore

	if !includeIgnored {
		gitignorePath := filepath.Join(rootPath, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			ignore, _ = gitignore.NewFromFile(gitignorePath)
		}
	}

	root := &FileNode{
		Name:  filepath.Base(rootPath),
		Path:  rootPath,
		IsDir: true,
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Calculate depth
		if maxDepth >= 0 {
			relPath, _ := filepath.Rel(rootPath, path)
			depth := len(strings.Split(relPath, string(filepath.Separator))) - 1
			if relPath == "." {
				depth = 0
			}

			// Skip if we've exceeded max depth
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip hidden directories (starting with .)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != rootPath {
			return filepath.SkipDir
		}

		// Check gitignore
		if !includeIgnored && ignore != nil {
			relPath, err := filepath.Rel(rootPath, path)
			if err == nil && relPath != "." {
				// Safely check if file should be ignored
				if ignore.Ignore(relPath) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		// Only include markdown files and directories
		if !d.IsDir() && !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Skip the root directory itself
		if path == rootPath {
			return nil
		}

		// Add to tree
		addToTree(root, rootPath, path, d.IsDir())

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort children at each level
	sortTree(root)

	return root, nil
}

func addToTree(root *FileNode, basePath, fullPath string, isDir bool) {
	relPath, _ := filepath.Rel(basePath, fullPath)
	parts := strings.Split(relPath, string(filepath.Separator))

	current := root
	for i, part := range parts {
		found := false
		for _, child := range current.Children {
			if child.Name == part {
				current = child
				found = true
				break
			}
		}

		if !found {
			newNode := &FileNode{
				Name:  part,
				Path:  filepath.Join(basePath, strings.Join(parts[:i+1], string(filepath.Separator))),
				IsDir: isDir || i < len(parts)-1,
			}
			current.Children = append(current.Children, newNode)
			current = newNode
		}
	}
}

func sortTree(node *FileNode) {
	if node == nil || len(node.Children) == 0 {
		return
	}

	// Sort: directories first, then alphabetically
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir
		}
		return strings.ToLower(node.Children[i].Name) < strings.ToLower(node.Children[j].Name)
	})

	// Recursively sort children
	for _, child := range node.Children {
		sortTree(child)
	}
}

func FlattenTree(node *FileNode, prefix string, isLast bool) []string {
	var lines []string

	if node == nil {
		return lines
	}

	// Create the display line
	if prefix != "" {
		var line string
		if isLast {
			line = prefix[0:len(prefix)-4] + "└── "
		} else {
			line = prefix[0:len(prefix)-4] + "├── "
		}

		if node.IsDir {
			line += "[+] " + node.Name + "/"
		} else {
			line += "[-] " + node.Name
		}
		lines = append(lines, line)
	}

	// Update prefix for children
	var newPrefix string
	if prefix == "" {
		newPrefix = "    "
	} else if isLast {
		newPrefix = prefix[0:len(prefix)-4] + "    "
	} else {
		newPrefix = prefix[0:len(prefix)-4] + "│   "
	}

	// Process children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		childLines := FlattenTree(child, newPrefix+"    ", childIsLast)
		lines = append(lines, childLines...)
	}

	return lines
}

func CollectFiles(node *FileNode) []string {
	var files []string

	if node == nil {
		return files
	}

	if !node.IsDir {
		files = append(files, node.Path)
	}

	for _, child := range node.Children {
		files = append(files, CollectFiles(child)...)
	}

	return files
}
