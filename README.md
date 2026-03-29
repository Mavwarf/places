# places

A CLI tool that bookmarks directories with shortcut names for quick navigation between projects.

## Why

Switching between project directories means typing long paths or hunting through `cd` history. `places` lets you save directories once and jump to them instantly with `p <name>`.

- Save any directory with a shortcut name — `p add api` in your API project
- Jump back instantly from anywhere — `p api`
- Fuzzy matching — `p ap` also works if there's only one match
- Interactive picker — just type `p` to browse all saved places with arrow keys
- Usage tracking — see which directories you visit most often

Works on Windows (PowerShell, cmd.exe) and macOS/Linux (Bash, Zsh). Includes a desktop app for managing places with one-click terminal launching. No external dependencies (CLI) / Wails v2 (desktop app).

## Usage

```
p                        # Browse saved places interactively and cd
p <name>                 # Jump to a saved place (supports fuzzy matching)
p add [name] [path]      # Save current dir (name auto-derived if omitted)
                         # Names: letters, numbers, hyphens, underscores, dots
p list                   # List all places with colored output and usage stats
p list --fav             # List only favorite places
p list --json            # List all places as JSON
p rm <name>              # Remove a saved place
p rename <old> <new>     # Rename a place (alias: mv)
p fav <name>             # Mark a place as favorite
p unfav <name>           # Unmark a place as favorite
p tag <name> <tag>       # Add a tag to a place
p untag <name> <tag>     # Remove a tag from a place
p tags                   # List all tags with place counts
p stats                  # Show usage summary
p app                    # Open the desktop app
p desktop <name> <0-4>   # Set virtual desktop for a place (0 = none)
p code <name>            # Open a place in VS Code
p shell <name>           # Open a new terminal at a place (no hook needed)
p where                  # Print the place name for the current directory
p exists <name>          # Exit 0 if a place exists, 1 otherwise (for scripts)
p prune                  # Remove places where the directory no longer exists
p note <name> [text...]   # Set a note (omit text to print, --rm to clear)
p export                 # Export all places and actions as JSON to stdout
p import <file>          # Import places and actions from JSON (skip existing)
p action add <name>      # Define a custom action (--label, --cmd required)
p action rm <name>       # Remove a custom action
p action list            # List all defined actions
p action assign <p> <a>  # Show action button on a place
p action unassign <p> <a> # Hide action button from a place
p autostart [on|off]     # Enable/disable starting tray app on login (Windows)
p edit [editor]          # Open places.json in $EDITOR or specified editor
p init                   # One-command setup (installs shell hooks)
p help                   # Show help
```

### Notes

Attach a text description to any place. Notes appear as subtitles in the desktop app dashboard.

```
p note api "billing REST API"   # Set a note
p note api                      # Print the note
p note api --rm                 # Clear the note
p list --json                   # Notes included in JSON output
```

In the desktop app, click a note to edit it inline (Enter to save, Escape to cancel). Hover over a place without a note to see an "add note" prompt.

### Import/Export

Back up or sync your places and actions across machines:

```
p export > backup.json          # Export all places + actions to JSON
p import backup.json            # Import from JSON (skips existing places/actions)
```

The export format matches `places.json` — it's the same structure, making it trivially round-trippable. Import uses a merge strategy: only places and actions that don't already exist are added.

The desktop app has Export and Import buttons in the sort bar for one-click backup/restore.

### Custom Actions

Define reusable shell commands and assign them to specific places. Custom action buttons appear in the desktop app and system tray alongside the built-in buttons, only on places they're assigned to. Right-click a custom action button in the dashboard to unassign it from that place. Built-in action buttons (cl, dir, VS, PS, >_) can be hidden per place via the **⋯** menu or by right-clicking the button.

`{path}` is replaced with the place's directory, `{name}` with the place's shortcut name. On Windows, commands run via `cmd /c`; on Unix, via `sh -c`. GUI apps need `start ""` on Windows to launch properly.

```
p action list                    # List all defined actions
p action assign api rider        # Show "rider" button on "api" place
p action unassign api rider      # Remove it
p action rm rider                # Delete action (also unassigns from all places)
```

**Adding actions:** PowerShell mangles embedded quotes when passing them to external programs. The easiest way to define actions with complex commands is to edit `places.json` directly (`p edit`). The `actions` section uses this format:

```json
{
  "actions": {
    "rider": {
      "label": "JR",
      "cmd": "start \"\" \"C:\\Program Files\\JetBrains\\Rider\\bin\\rider64.exe\" \"{path}\""
    },
    "godot": {
      "label": "GD",
      "cmd": "start \"\" \"C:\\dev\\tools\\godot\\4.6.1\\Godot_v4.6.1-stable_win64.exe\" -e --path \"{path}\""
    }
  }
}
```

For simple commands without spaces in paths, the CLI works fine:

```
p action add mytest --label "T" --cmd "echo {name} at {path}"
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

Tags are lowercase, deduplicated, and sorted alphabetically. The desktop app shows tag badges on each place with click-to-add and click-to-remove, plus a filter bar with toggleable tag chips (select multiple tags to show places matching any of them).

### Favorites

Mark frequently-used places for quick filtering:

```
p fav api                # mark as favorite
p unfav api              # unmark
p list --fav             # show only favorites
p list --json --fav      # filtered JSON output
```

Favorites show a ★ marker in `p list`. The desktop app has a clickable star toggle per place and a ★ chip in the filter bar.

## Desktop App

`p app` launches a desktop GUI (built with Wails v2) that shows all saved places with action buttons:

- **cl** — continue last Claude Code session at that directory (Shift+click for fresh session with confirmation, Ctrl+click / Cmd+click for YOLO mode with `--dangerously-skip-permissions`)
- **dir** / **📁** — open Explorer (Windows) or Finder (macOS) at that directory
- **VS** — open VS Code at that directory
- **PS** / **>_** — open PowerShell/cmd (Windows) or Terminal/iTerm2 (macOS) at that directory
- **⋯** — place menu: toggle built-in action visibility, assign/unassign custom actions, remove place
- **Recent** — toggleable bar showing last 8 launched place+action pairs as quick-replay chips

Each place row is organized into aligned columns: **name** (fixed width, ellipsis for long names), **path / note** (flexible, fills available space), **stats** (usage count + last-used time), **git + tags** (status badge, tag badges, add-tag button), and **action buttons** (right-aligned). All columns have fixed widths so they align consistently across rows.

Click a place name to rename it inline, or click a path to edit it (Enter to save, Escape to cancel). Notes appear as a subtitle below the path and are truncated with ellipsis when too long; click to edit inline, or hover over a place without a note to add one. Export and Import buttons in the sort bar allow one-click backup/restore. The **git + tags** column shows the current branch badge (green ✓ for clean, yellow ● for dirty, red for errors) followed by a **git** button to fetch/refresh status. Git status is fetched automatically for all places on startup; use the **Update git** button to refresh manually. A **Git dirty** filter chip in the filter bar shows only places with uncommitted changes. Right-click any built-in action button to hide it from that place; use the **⋯** menu to show it again.

Each place also has a virtual desktop selector (D1–D*N*, count detected automatically). When set, the app switches to that desktop before launching any tool. A **→** button next to the selector lets you jump to that desktop without launching anything. Uses `VirtualDesktopAccessor.dll` (place it next to `places-app.exe`).

Press **Win+Alt+P** from anywhere to open the dashboard. If it's on another virtual desktop, you are switched there automatically.

The place list is sorted by last used (most recent on top) by default, with relative timestamps (e.g. "2h ago"). Can also sort by name, most used, date added, frecency (frequency × recency), or desktop (grouped by virtual desktop, unassigned at bottom). Places with equal sort values use alphabetical name as a stable tiebreaker. Each place has a clickable star to toggle its favorite status. Filter and sort state persists across restarts.

A **Compact** toggle in the header switches to a dense single-line view showing name, tags, stats, and action buttons. Typing anywhere auto-focuses the filter input for quick search.

24 color themes grouped by category: **Dark** (Dark, Nord, Dracula, One Dark, Tokyo Night, Catppuccin, Rosé Pine, Kanagawa, Everforest, Monokai, Palenight, Poimandres, Synthwave, Vesper, Ayu Dark, Flexoki), **Warm** (Gruvbox, Solarized), **Light** (Light, Catppuccin Latte, Rosé Dawn, Everforest Light, Ayu Light, Flexoki Light). Click to cycle, right-click to restore startup theme, or use the **...** dropdown picker with live preview on hover.

A fixed status bar at the bottom of the window shows author/GitHub/Wiki links on the left, a place count in the center (e.g. "5 / 12 places" when filtered), and the build version on the right — always visible regardless of scroll position.

The header section (title, sort bar, filter bar) stays fixed at the top while the place list scrolls independently. The add form is hidden by default — click the **+** button in the header to expand it. After adding a place, the form auto-collapses. Click the **…** button next to the path input to open a native folder picker, or drag a folder from Explorer onto the window to fill in the path. Changes are shared with the CLI (same `places.json`).

### Tags in the App

The desktop app supports tags with:
- **Tag badges** on each place — click **x** to remove a tag
- **+** button on each place — click to add a tag via prompt
- **Filter bar** — click a tag chip to include places with that tag; right-click to exclude; type in the text filter to search by name, path, or note; click "Clear" to reset all filters
- **Tags input** in the add form — comma-separated tags when adding a new place

### Screenshot Mode

Press **S** to toggle screenshot mode. This anonymizes place names with fantasy names for taking screenshots without exposing real project names. Places tagged with `work` also get their directory paths anonymized (keeping the `C:\dev\repos\` prefix). Mappings persist in localStorage so names stay consistent across reloads.

### Color Themes

Click the theme toggle button (top-right) to cycle through 24 color themes, or use the **...** dropdown picker grouped by Dark, Warm, and Light categories with live preview on hover. Your choice is saved in localStorage and persists across sessions.

### Window Controls

The header includes **pin** (📌), **minimize** (−), and **quit** (×) buttons in the top-right corner. Pin toggles always-on-top mode (the window stays above all other windows); the state persists across restarts. Minimize hides the window; quit fully exits the app. You can also close the window normally to hide to tray (Shift+close to exit).

The window remembers its position and size between sessions (saved to `~/.config/places/window.json`). On first launch it uses the default 1100×600 size. The minimum width is 1000px to ensure all columns display correctly.

### System Tray

The desktop app lives in the system tray. Closing the window hides it to the tray instead of exiting — the app stays running for quick access.

- **Double-click** the tray icon to reopen the dashboard
- **Left-click** or **right-click** for the context menu:
  - **Open Dashboard** — show the main window
  - **Place submenus** — each saved place has PowerShell, Claude, cmd, and Explorer actions
  - **Refresh** — reload places from config
  - **Quit** — fully exit the app

### Build

**Windows:**
```
cd cmd/places-app
go build -tags production -ldflags "-X main.version=$(git describe --tags --always) -X 'main.buildTime=$(date -u +%Y-%m-%d\ %H:%M\ UTC)' -H windowsgui" -o places-app.exe .
```

Copy `places-app.exe` next to `places.exe` on your PATH.

**macOS:**
```
make mac
```

This builds the CLI and desktop app, installs to `~/.local/bin`, and creates `/Applications/Places.app` (launchable via Spotlight). Requires Xcode command line tools.

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
  "actions": {
    "git": {
      "label": "git",
      "cmd": "cmd /c start powershell -NoExit -Command \"Set-Location '{path}'; git status\""
    }
  },
  "places": {
    "api": {
      "path": "/home/user/projects/api",
      "added_at": "2026-03-01T13:50:17+01:00",
      "use_count": 5,
      "last_used_at": "2026-03-01T14:10:42+01:00",
      "tags": ["backend", "work"],
      "desktop": 2,
      "actions": ["git"],
      "note": "billing REST API",
      "hidden_defaults": ["cmd"]
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
