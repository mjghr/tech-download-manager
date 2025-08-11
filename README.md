# Tech Download Manager

A high-performance file download manager built with Go, featuring concurrent downloads, pause/resume functionality, and advanced queue management.

[![Go Version](https://img.shields.io/badge/Go-1.23.4-blue.svg)](https://golang.org)

## Features

- **Concurrent Downloads**: Split files into chunks and download them concurrently for maximum speed
- **Pause/Resume**: Pause and resume downloads at any time without losing progress
- **Speed Limiting**: Control bandwidth usage with configurable speed limits
- **Queue Management**: Manage multiple downloads with configurable concurrency limits
- **Real-time Progress**: Track download progress with detailed statistics
- **Scheduled Downloads**: Run downloads within specific time windows
- **Temporary Files**: Use temporary files for resumable downloads with automatic cleanup
- **Error Handling**: Automatic retry of failed chunks with graceful error handling
- **Modern TUI**: Beautiful terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)

## Installation

### Prerequisites
- Go 1.23.4 or higher
- Git

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/mjghr/tech-download-manager.git
cd tech-download-manager
```

2. Build and run:
```bash
go run cmd/main.go
```

## Usage

1. Launch the application using the command above
2. Use the intuitive TUI interface to:
   - Add new downloads
   - Manage download queues
   - Monitor progress
   - Configure settings

## Project Structure

```
.
├── cmd/           # Main application entry point
├── client/        # HTTP client implementation
├── config/        # Configuration management
├── controller/    # Business logic and controllers
├── manager/       # Download manager implementation
├── models/        # Data models and structures
├── ui/            # Terminal user interface components
└── util/          # Utility functions and helpers
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal UI

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.



