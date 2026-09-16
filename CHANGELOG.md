# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.1.0] - 2026-09-16

### 🚀 Added

- **Pure-Go Windows Console Support**: Native raw terminal and console mode on Windows using `kernel32.dll` system calls via standard library `syscall` (zero CGO, zero third-party dependencies).
- **Non-TTY & Pipe Fallback**: Seamless fallback to standard buffered reading when stdin/stdout are redirected or piped (`echo "status" | cli`).
- **Dynamic Autocompletion & Column Grid**:
  - Extensible `Completer` interface: `Complete(ctx, line, pos)`.
  - Built-in `PrefixCompleter` and `PathCompleter`.
  - Terminal-width-aware adaptive multi-column grid layout with descriptions.
  - Candidate cycling via `Tab` (forward) and `Shift+Tab` / `KeyBackTab` (backward).
- **Token & Quote Parser**: Lexes single quotes (`'...'`), double quotes (`"..."`), and backslash-escaped spaces (`\ `) for argument completion boundaries.
- **Reverse Incremental Search (Ctrl+R / Ctrl+S)**: Interactive history search with real-time matching, query editing via backspace, and safe cancel without input buffer corruption.
- **Password & Secret Mode**: `ReadPassword(prompt...)` reading secrets without terminal echo (or optional mask character) and without saving to history.
- **Context-Aware ReadLine**: `ReadLineContext(ctx, prompt...)` enabling non-blocking timeouts and cancellation deadlines.
- **Emacs-Style Kill Ring**: Circular kill ring saving deletions from `Ctrl+K`, `Ctrl+U`, `Ctrl+W`, and `Alt+D` with yank (`Ctrl+Y`) and yank-pop (`Alt+Y`).
- **Multi-Level Undo & Redo**: `UndoStack` with undo (`Ctrl+_`) and redo support.
- **Word Navigation & Editing**:
  - Word left: `Alt+B` / `Ctrl+Left`
  - Word right: `Alt+F` / `Ctrl+Right`
  - Delete word after: `Alt+D`
  - Transpose characters: `Ctrl+T`
- **Bracketed Paste**: Atomic insertion of pasted text blocks without triggering premature command execution.
- **Advanced Unicode Display Width**: Full classification of zero-width runes (combining marks, zero-width joiners `0x200D`, zero-width spaces, variation selectors) as width 0, and wide CJK/emojis as width 2.
- **Customizable KeyMap**: Extensible keybinding configuration table via `Config.KeyMap`.
- **History Compaction**: `History.CompactFile(path)` for bounded on-disk history persistence.
- **Dedicated Examples Suite**: Reference implementations under `examples/` for `basic`, `history`, `completion`, `password`, `context`, and `custom-bindings`.

### 🐛 Fixed

- **Signal Safety & Terminal Restore**: Prevented signal watcher from calling `os.Exit(1)` on SIGINT; eliminated goroutine leak on editor shutdown.
- **Panic Recovery**: Added deferred raw mode cleanup in `ReadLine` and `ReadPassword` to guarantee shell restoration even if a panic occurs.
- **Multi-Row Line Wrapping**: Replaced single-row `\r\033[K` assumption with multi-row cursor tracking and clearance across terminal column boundaries.
- **Escape Key Buffer Loss**: Fixed parser state machine dropping subsequent bytes when a lone `Esc` is entered.
- **Completion Candidate Cycling**: Fixed candidate string concatenation during repeated `Tab` presses by resetting buffer to initial state before each candidate insertion; pressing `Esc` restores original buffer.

### 🧪 Tests & Quality

- Added virtual terminal tests and comprehensive feature coverage.
- Added 3 Go fuzz targets: `FuzzEscapeParser`, `FuzzParseTokens`, and `FuzzRuneDisplayWidth`.
- Added performance benchmarks for buffer insertion, display width, history search, token parsing, and redraw.
- 100% race-clean test suite across all packages (`go test -race ./...`).

---

## [v1.0.0] - 2026-07-15

### 🚀 Initial Release

- Core pure-Go line editor for Linux and macOS.
- Basic in-memory and persistent file history with deduplication.
- Prefix search via PageUp and PageDown.
- UTF-8 multi-byte rune decoding.
- Companion visual formatting subpackage `format`.
