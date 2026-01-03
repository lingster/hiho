# hiho

Terminal-first tmux companion built with Go, Bubble Tea, and Lip Gloss.

## Prerequisites
- Go 1.24+
- `tmux` available on `PATH`

## Install & Run
```bash
go run ./cmd/hiho
# or build a binary
go build -o bin/hiho ./cmd/hiho
```

## UI Layout

The TUI features a tabbed interface:
- **Tab bar** at the top with [Conversation] and [Tmux Window] tabs
- **Main content area** showing either conversation history or tmux session output
- **2-line input area** at the bottom with command help

Sessions are named `hiho-<pid>-<n>` where `<pid>` is the hiho process ID and `<n>` is an incrementing counter.

## Slash Commands

| Command | Description |
|---------|-------------|
| `/help` | Show available slash commands |
| `/new <cmd>` | Create a tmux session and run the command |
| `/list` | List all hiho-managed sessions |
| `/sessions` | List all tmux sessions |
| `/next` | Cycle to next session |
| `/prev` | Cycle to previous session |
| `/switch <session>` | Jump to a specific session |
| `/switch` | Cycle to next session (when in Tmux tab) |
| `/closeall` | Close all hiho-managed sessions |
| `/view tmux` | Switch to Tmux Window tab |
| `/view conversation` | Switch to Conversation tab |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Toggle between Conversation and Tmux Window tabs |
| `Alt+Left` / `Alt+h` | Previous session |
| `Alt+Right` / `Alt+l` | Next session |
| `Alt+Up` / `Alt+j` | Previous session |
| `Alt+Down` / `Alt+k` | Next session |
| `Ctrl+C` | Quit |

## Tests
```bash
go test ./...
```
The integration tests for tmux will be skipped automatically if `tmux` is not installed.
