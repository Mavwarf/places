# places

A CLI tool that bookmarks directories with shortcut names for quick navigation between projects.

## Why

Switching between project directories means typing long paths or hunting through `cd` history. `places` lets you save directories once and jump to them instantly with `p <name>`. It's especially useful when:

- You work across many repositories and want single-word shortcuts
- You start new terminal or Claude Code sessions and need to get to the right directory fast
- You want to see which project directories you visit most often

## Usage

```
p                        # Browse saved places interactively and cd
p select                 # Same as above
p <name>                 # Jump to a saved place (supports fuzzy matching)
p add [name] [path]      # Save current dir (name auto-derived if omitted)
p add <name> <path>      # Save a specific path
p list                   # List all places with colored output and usage stats
p rm <name>              # Remove a saved place
p rename <old> <new>     # Rename a place (alias: mv)
p stats                  # Show usage summary
p edit [editor]          # Open places.json in $EDITOR or specified editor
p init                   # One-command setup (installs shell hooks)
p help                   # Show help
```

### Example

```
cd C:\dev\repos\private\cli_tools\notify
p add notify

cd C:\dev\repos\private\cli_tools\places
p add places

p list
  notify  C:\dev\repos\private\cli_tools\notify  (added Feb 28, 3 uses, last: Feb 28 14:10)
  places  C:\dev\repos\private\cli_tools\places  (added Feb 28, 1 use, last: Feb 28 13:50)

p notify    # instantly cd to the notify project
p not       # fuzzy match — also jumps to notify
```

Running `p` or `p select` opens an interactive selector with arrow-key navigation:

```
  > notify  C:\dev\repos\private\cli_tools\notify
    places  C:\dev\repos\private\cli_tools\places
  ↑/↓ navigate, Enter select, Esc cancel
```

## How it works

A child process cannot change the parent shell's working directory. `places` solves this by splitting the work:

- The `places` binary handles storage and retrieval (add, list, go, rm)
- A shell function `p()` wraps the binary, captures the path from `places go`, and performs the actual `cd` / `Set-Location`

The `places shell-hook install` command injects this `p()` function into your shell config file using marker comments (`# BEGIN places shell-hook` / `# END places shell-hook`) for clean install and uninstall. For cmd.exe, it creates a `p.bat` batch file next to `places.exe`.

## Setup

### Requirements

- Go 1.24+ (to build from source)
- Bash, Zsh, PowerShell, or cmd.exe

### Quick setup

After building and placing the binary on your PATH:

```
places init
```

This auto-detects your shell, installs the `p` hook, and on Windows also creates `p.bat` for cmd.exe. Follow the printed next steps to reload your profile.

### Build

```
cd cmd/places
go build -o places.exe .
```

Copy the binary somewhere on your `PATH`:

```
cp places.exe C:\dev\tools\cli\    # or /usr/local/bin/ on Linux/macOS
```

### Shell integration

#### Bash / Zsh

```
places shell-hook install
source ~/.bashrc    # or source ~/.zshrc
```

#### PowerShell

PowerShell requires script execution to be enabled (one-time):

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

Then install the hook:

```powershell
places shell-hook install
. $PROFILE
```

#### cmd.exe

```
places shell-hook install --shell cmd
```

This creates a `p.bat` next to `places.exe`. No restart needed — works immediately in any new cmd window.

#### Multiple shells

Use `--shell` to install for a specific shell:

```
places shell-hook install --shell bash
places shell-hook install --shell powershell
places shell-hook install --shell cmd
```

### Uninstall

```
places shell-hook uninstall
places shell-hook uninstall --shell bash
```

## Storage

Places are stored in `~/.config/places/places.json` with usage statistics:

```json
{
  "places": {
    "notify": {
      "path": "C:\\dev\\repos\\private\\cli_tools\\notify",
      "added_at": "2026-02-28T13:50:17+01:00",
      "use_count": 3,
      "last_used_at": "2026-02-28T14:10:42+01:00"
    }
  }
}
```

## License

MIT
