# places

A CLI tool that bookmarks directories with shortcut names for quick navigation between projects.

## Why

Switching between project directories means typing long paths or hunting through `cd` history. `places` lets you save directories once and jump to them instantly with `p <name>`.

- Save any directory with a shortcut name — `p add api` in your API project
- Jump back instantly from anywhere — `p api`
- Fuzzy matching — `p ap` also works if there's only one match
- Interactive picker — just type `p` to browse all saved places with arrow keys
- Usage tracking — see which directories you visit most often

Works on Windows (PowerShell, cmd.exe) and Unix (Bash, Zsh). Includes a desktop app for managing places with one-click terminal launching. No external dependencies (CLI) / Wails v2 (desktop app).

## Usage

```
p                        # Browse saved places interactively and cd
p <name>                 # Jump to a saved place (supports fuzzy matching)
p add [name] [path]      # Save current dir (name auto-derived if omitted)
p list                   # List all places with colored output and usage stats
p list --json            # List all places as JSON
p rm <name>              # Remove a saved place
p rename <old> <new>     # Rename a place (alias: mv)
p tag <name> <tag>       # Add a tag to a place
p untag <name> <tag>     # Remove a tag from a place
p tags                   # List all tags with place counts
p stats                  # Show usage summary
p app                    # Open the desktop app
p code <name>            # Open a place in VS Code
p shell <name>           # Open a new terminal at a place (no hook needed)
p where                  # Print the place name for the current directory
p exists <name>          # Exit 0 if a place exists, 1 otherwise (for scripts)
p prune                  # Remove places where the directory no longer exists
p autostart [on|off]     # Enable/disable starting tray app on login (Windows)
p edit [editor]          # Open places.json in $EDITOR or specified editor
p init                   # One-command setup (installs shell hooks)
p help                   # Show help
```

### Example

```
cd ~/projects/api
p add api --tag work --tag backend

cd ~/projects/frontend
p add

p list
  api       ~/projects/api       (added Mar 1, 5 uses, last: Mar 1 14:10)
  frontend  ~/projects/frontend  (added Mar 1, 2 uses, last: Mar 1 13:50)

p api         # instantly cd to the api project
p ap          # fuzzy match — also works
```

Running `p` opens an interactive selector with arrow-key navigation:

```
  > api       ~/projects/api
    frontend  ~/projects/frontend
  ↑/↓ navigate, Enter select, Esc cancel
```

### Tags

Organize places with tags for filtering:

```
p add api --tag work --tag backend    # tag on creation
p tag api devops                      # add a tag later
p untag api devops                    # remove a tag
p list --tag work                     # filter by tag
p list --json --tag work              # filtered JSON output
p tags                                # list all tags with counts
```

Tags are lowercase, deduplicated, and sorted alphabetically. The desktop app shows tag badges on each place with click-to-add and click-to-remove, plus a filter bar to show only places with a specific tag.

## Desktop App

`p app` launches a desktop GUI (built with Wails v2) that shows all saved places with action buttons:

- **PS** — open PowerShell at that directory
- **Claude** — open PowerShell at that directory and start Claude Code
- **cmd** — open cmd.exe at that directory
- **dir** — open Explorer at that directory

The place list is sorted by last used (most recent on top) by default, with relative timestamps (e.g. "2h ago"). Can also sort by name, most used, or date added.

You can also add and remove places from the app. The **…** button next to the path input opens a native folder picker dialog. Changes are shared with the CLI (same `places.json`).

### Tags in the App

The desktop app supports tags with:
- **Tag badges** on each place — click **x** to remove a tag
- **+** button on each place — click to add a tag via prompt
- **Filter bar** — click a tag chip to show only places with that tag, click "All" to reset
- **Tags input** in the add form — comma-separated tags when adding a new place

### Screenshot Mode

Press **S** to toggle screenshot mode. This anonymizes place names with fantasy names for taking screenshots without exposing real project names. Places tagged with `work` also get their directory paths anonymized (keeping the `C:\dev\repos\` prefix). Mappings persist in localStorage so names stay consistent across reloads.

### Color Themes

Click the theme toggle button (top-right) to cycle through 6 color schemes: Dark (default), Light, Nord, Dracula, Solarized, and Gruvbox. Your choice is saved in localStorage and persists across sessions.

### Window Controls

The header includes **minimize** (−) and **quit** (×) buttons in the top-right corner. Minimize hides the window; quit fully exits the app. You can also close the window normally to hide to tray (Shift+close to exit).

### System Tray

The desktop app lives in the system tray. Closing the window hides it to the tray instead of exiting — the app stays running for quick access.

- **Double-click** the tray icon to reopen the dashboard
- **Left-click** or **right-click** for the context menu:
  - **Open Dashboard** — show the main window
  - **Place submenus** — each saved place has PowerShell, Claude, cmd, and Explorer actions
  - **Refresh** — reload places from config
  - **Quit** — fully exit the app

### Build

```
cd cmd/places-app
go build -tags production -o places-app.exe .
```

Copy `places-app.exe` next to `places.exe` on your PATH.

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

Build and place the binary on your PATH, then run:

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
cp places.exe /usr/local/bin/       # Linux/macOS
cp places.exe C:\dev\tools\cli\     # Windows
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
    "api": {
      "path": "/home/user/projects/api",
      "added_at": "2026-03-01T13:50:17+01:00",
      "use_count": 5,
      "last_used_at": "2026-03-01T14:10:42+01:00",
      "tags": ["backend", "work"]
    }
  }
}
```

## Author

Created by Thomas Häuser
- https://mavwarf.netlify.app/
- https://github.com/Mavwarf/places

## License

MIT
