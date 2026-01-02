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

The TUI opens with a chat-style view and a prompt at the bottom.

## Slash Commands
- `/new <cmd>`: create a tmux session and run the command (e.g., `/new echo hello world`).
- `/next` and `/prev`: cycle through tmux sessions.
- `/switch <session>`: jump to a specific session.
- `/sessions`: list known tmux sessions.
- `/view session` or `/view conversation`: toggle between session output and chat history.

Session output is appended to the conversation log and can be focused in the session view.

## Tests
```bash
go test ./...
```
The integration test for tmux will be skipped automatically if `tmux` is not installed.
