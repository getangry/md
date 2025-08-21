# md - Terminal Markdown Viewer

A terminal-based markdown viewer built with Bubble Tea that provides beautiful markdown rendering and file navigation.

## Features

- Beautiful markdown rendering with syntax highlighting
- File tree navigation for markdown files  
- Single file viewing with paging (like `less`)
- Respects `.gitignore` by default
- Advanced ANSI/ASCII styling for rich text rendering
- Intuitive keyboard controls
- **Instant startup** - UI appears immediately (zero blocking operations)
- **Lazy file loading** - files read asynchronously after UI initialization  
- **Lazy rendering** - markdown renderer created only when needed
- **Progressive file discovery** - starts with current directory, expands deeper automatically
- **Background processing** - all I/O operations happen in background threads
- **Graceful error handling** - file errors displayed in UI without crashes

## Usage

### View a single file
```bash
md README.md
```

### Read from stdin (pipe support)
```bash
cat file.md | md
echo "# Hello World" | md
curl -s https://example.com/readme.md | md
```

### Browse and view multiple files
```bash
md
```

### Include files from .gitignore
```bash
md -i
```

## Keyboard Controls

### Single File Mode
- `q`, `Ctrl+C`, `Esc`: Quit
- `j`, `↓`: Scroll down one line
- `k`, `↑`: Scroll up one line
- `Ctrl+D`, `PgDn`: Scroll down half page
- `Ctrl+U`, `PgUp`: Scroll up half page
- `Space`: Scroll down one page
- `g`, `Home`: Go to top
- `G`, `End`: Go to bottom
- `r`: Toggle raw/rendered view

### Dual Pane Mode
- `Tab`: Switch focus between tree and content panes
- `h`, `←`: Focus tree pane
- `l`, `→`: Focus content pane
- `j`, `↓`: Navigate down (tree) or scroll down (content)
- `k`, `↑`: Navigate up (tree) or scroll up (content)
- `Enter`: Select file and focus content pane
- `<`, `{`: Decrease tree pane width
- `>`, `}`: Increase tree pane width
- `e`: Manually expand to scan deeper directories  
- `r`: Toggle raw/rendered view
- `q`, `Ctrl+C`: Quit

## Installation

### Using Make (Recommended)
```bash
make build      # Build for current platform
make install    # Install to $GOPATH/bin
make test       # Run tests
make coverage   # Generate coverage report
```

### Cross-Platform Builds
Build for multiple platforms and architectures:
```bash
make build-all      # Build for all platforms
make build-linux    # Build for Linux (amd64 + arm64)
make build-windows  # Build for Windows (amd64 + arm64)  
make build-darwin   # Build for macOS (amd64 + arm64)
make release        # Create release archives
```

Binaries are output to:
- `build/` - Local development builds
- `dist/linux/`, `dist/windows/`, `dist/darwin/` - Cross-platform builds
- `dist/archives/` - Release archives (`.tar.gz` for Unix, `.zip` for Windows)

### Manual Build
```bash
go build -o md
```

## Development

```bash
make check      # Run format, vet, and tests
make dev        # Build with race detection
make release    # Build optimized binary
make clean      # Clean build artifacts
```

## Testing

Run the comprehensive test suite:
```bash
make test       # Run all tests
make coverage   # Generate HTML coverage report
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [go-gitignore](https://github.com/denormal/go-gitignore) - Gitignore support

## License

MIT License - see [LICENSE](LICENSE) file for details.