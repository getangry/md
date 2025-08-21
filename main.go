package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	inclusive bool
)

func init() {
	flag.BoolVar(&inclusive, "i", false, "Include files in .gitignore")
}

func main() {
	flag.Parse()
	args := flag.Args()

	var m tea.Model
	var err error

	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	hasStdin := (stat.Mode() & os.ModeCharDevice) == 0

	if hasStdin {
		// Stdin mode - read from pipe
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		m, err = NewSingleFileModelWithContent("stdin", string(content))
		if err != nil {
			fmt.Printf("Error creating stdin viewer: %v\n", err)
			os.Exit(1)
		}
	} else if len(args) > 0 {
		// Single file mode
		filename := args[0]
		m, err = NewSingleFileModel(filename)
		if err != nil {
			fmt.Printf("Error loading file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Directory tree mode
		m, err = NewDualPaneModel(inclusive)
		if err != nil {
			fmt.Printf("Error initializing: %v\n", err)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
