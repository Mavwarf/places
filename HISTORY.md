# History

## Features

- Virtual desktop (`places desktop <name> <0-4>`) — assign a Windows virtual desktop to a place; dashboard and tray switch to that desktop before launching tools; uses VirtualDesktopAccessor.dll *(Mar 2)*
- Auto-refresh in desktop app — polls /api/places every 3 seconds so CLI-added places appear automatically *(Mar 2)*
- Usage tracking from app/tray — clicking action buttons (PS, cl, VS, >_, dir) now increments use count and updates last-used timestamp; UI refreshes immediately *(Mar 2)*
- Favorites (`places fav`/`unfav`) — mark places as favorites; filter with `--fav` in list and `--json`; star toggle per place and ★ filter button in desktop app *(Mar 1)*
- Interactive arrow-key select — `places select` uses cursor navigation instead of numbered input; Up/Down to move, Enter to confirm, Esc to cancel *(Mar 1)*
- Init command (`places init`) — one-command setup that installs shell hooks for detected shell + cmd on Windows *(Mar 1)*
- Edit command (`places edit [editor]`) — open places.json in `$EDITOR` or specified editor *(Mar 1)*
- Fuzzy matching — `p not` matches `notify` via substring; resolves if exactly one match *(Mar 1)*
- Color output — `places list` uses ANSI colors: green names, cyan paths, dim stats, yellow warnings *(Mar 1)*
- Path separator normalization — all paths normalized to OS-native separators (`\` on Windows) on load *(Mar 1)*
- Relative path resolution — `places add name .` resolves `.` and `..` to absolute paths *(Mar 1)*
- Select sorted by recent use — most recently used places shown first in `places select` *(Mar 1)*
- Usage stats (`places stats`) — total uses, most/least used place summary *(Mar 1)*
- Rename command (`places rename`/`mv`) — rename a place while preserving stats *(Mar 1)*
- Path validation — `list` and `select` show `[missing!]` for deleted directories *(Mar 1)*
- Auto-name on add — `places add` with no name derives it from the directory basename *(Mar 1)*
- cmd.exe support — `places shell-hook install --shell cmd` creates `p.bat` next to `places.exe`; uses temp file for interactive select *(Mar 1)*
- CLAUDE.md — project conventions, build/deploy instructions, shell hook update workflow *(Mar 1)*
- Shell hook passthrough — `p add/rm/list/help/...` passes through to `places` binary; `p`/`p select` both do select+cd *(Feb 28)*
- PowerShell support — shell hook install/uninstall for PowerShell (Core and Windows PowerShell); `--shell` flag for targeting specific shells *(Feb 28)*
- Interactive select (`places select`) — numbered menu to browse and pick a place; prints path to stdout for shell wrapper *(Feb 28)*
- Usage statistics — per-place tracking of added_at, use_count, last_used_at; shown in `places list` output *(Feb 28)*
- Config format migration — auto-migrates old `map[string]string` format to new `map[string]*Place` on load *(Feb 28)*
- Shell hook (`places shell-hook`) — marker-based install/uninstall of `p()` function into `.bashrc`/`.zshrc`/PowerShell profile *(Feb 28)*
- Core commands — `add`, `list`/`ls`, `go`, `rm` for managing directory bookmarks *(Feb 28)*
- Initial release — Go CLI tool for bookmarking directories with shortcut names *(Feb 28)*

---

## 2026-03-02

### Virtual desktop support

Each place can be assigned a Windows virtual desktop (1–4) via `places desktop
<name> <n>`. When launching a tool (PowerShell, Claude, cmd, VS Code, Explorer)
from the desktop app or system tray, the app switches to that desktop first using
`VirtualDesktopAccessor.dll`.

- CLI: `p desktop api 2` assigns desktop 2, `p desktop api 0` clears it
- `p list` shows a `[D2]` badge for places with a desktop set
- `p list --json` includes `"desktop"` field
- Desktop app: dropdown selector (—, D1–D4) per place row
- System tray: submenus switch desktop before launching
- New `internal/desktop` package (copied from notify tool)
- `launcher.SwitchDesktop(n)` helper called before `Detach()`

### Auto-refresh in desktop app

The dashboard now polls `/api/places` every 3 seconds. Places added via CLI
appear automatically without needing to interact with the app.

### Usage tracking from app and tray

Clicking any action button (PS, cl, VS, >_, dir) in the desktop app or system
tray now records a use — increments `use_count` and updates `last_used_at`. The
UI refreshes immediately after clicking, so the use count and sort order update
in place.

## 2026-03-01

### Interactive arrow-key select

`places select` (and `p` / `p select`) now uses an interactive cursor-based menu
instead of numbered input. Arrow keys move the highlight, Enter confirms, Esc or
`q` cancels. The selected line shows bold green name and cyan path; other lines
are dimmed. A footer hint shows available keys. The menu cleans up after itself
(all lines cleared on exit).

Implementation uses platform-specific raw terminal input: `ReadConsoleInputW`
with virtual key codes on Windows, termios with VT100 escape sequences on Unix.
No external dependencies.

### Init command

`places init` is a one-command setup that auto-detects the current shell,
installs the shell hook, and on Windows also installs `p.bat` for cmd.exe.
Skips hooks that are already installed (no error). Prints next steps
(execution policy, profile reload).

### Edit command

`places edit [editor]` opens `places.json` in an editor. Priority:
explicit argument > `$EDITOR` > `$VISUAL` > `notepad` (Windows) / `vi` (Unix).
Example: `places edit notepad`, `places edit code`.

### Fuzzy matching

`places go` (and `p <name>`) now falls back to substring matching when no
exact match is found. `p not` matches `notify` if it's the only place
containing "not". Ambiguous matches (multiple results) still show an error.

### Color output

`places list` now uses ANSI color codes: place names in green, paths in cyan,
stats in dim, and `[missing!]` warnings in yellow.

### Path separator normalization

All stored paths are normalized to OS-native separators on load via
`filepath.Clean()`. On Windows, forward slashes are converted to backslashes.
Existing paths with mixed separators are fixed automatically on the next save.

### Relative path resolution

`places add name .` and similar relative paths are now resolved to absolute
paths via `filepath.Abs()` before saving. Previously `.` was stored literally.

### Select sorted by recent use

`places select` now shows places sorted by most recently used first, with
never-used places at the bottom (sorted alphabetically). `places list` remains
alphabetical.

### Usage stats

`places stats` shows a quick summary:

```
Places: 5
Total uses: 15
Most used: notify (12 uses)
Least used: eco (0 uses)
```

### Rename command

`places rename <old> <new>` (alias: `mv`) renames a saved place while
preserving all statistics (added_at, use_count, last_used_at).

### Path validation

`places list` and `places select` now check if each saved path still exists on
disk. Missing directories are flagged with `[missing!]` in the output.

### Auto-name on add

`places add` without a name argument derives the name from the current
directory's basename. E.g. running `places add` in `/cli_tools/notify` saves it
as `notify`.

### cmd.exe support

Added `cmd` as a supported shell type. `places shell-hook install --shell cmd`
creates a `p.bat` file next to the `places.exe` binary. The batch file handles:

- `p <name>` — `for /f` captures `places go` output and `cd /d` to it
- `p` / `p select` — runs `places select` with stdout redirected to a temp file
  (preserves stdin for interactive input), then reads and `cd`s to the result
- `p add/rm/list/...` — passthrough to `places`

Uninstall with `places shell-hook uninstall --shell cmd` (deletes `p.bat`).

### CLAUDE.md

Added project conventions file covering build/deploy, documentation rules, shell
hook update workflow, and coding conventions.

### go.mod version fix

Fixed `go 1.24.0` to `go 1.24` to match the required format.

## 2026-02-28

### Shell hook passthrough

The `p` function now acts as a full wrapper for `places`. Subcommands like `add`,
`rm`, `list`, `ls`, `help`, and `shell-hook` pass through directly to the binary.
`p` and `p select` both run the interactive selector and cd to the result.
Anything else is treated as a place name and resolved via `places go`.

### PowerShell support

Shell hook install/uninstall now supports PowerShell alongside bash/zsh. On
Windows, auto-detection defaults to PowerShell. The `--shell` flag allows
targeting a specific shell.

- PowerShell requires `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`
- PowerShell Core (pwsh) and Windows PowerShell have separate profile paths
- Both must be updated when the snippet changes

### Interactive select

`places select` shows a numbered list of saved places and prompts for a choice.
The menu and prompt go to stderr, the selected path goes to stdout — this lets
the shell wrapper capture the path while the user sees the menu.

### Usage statistics

Each place now tracks when it was added, how many times it has been used, and
when it was last used. Stats are shown inline in `places list`:

```
  notify  C:/dev/repos/private/cli_tools/notify  (added Feb 28, 3 uses, last: Feb 28 14:10)
  places  C:/dev/repos/private/cli_tools/places  (added Feb 28, never used)
```

Usage is recorded on every `places go` and `places select` call.

### Config format migration

The config format changed from `map[string]string` to `map[string]*Place` to
support per-place statistics. The old format auto-migrates on first load — plain
string values are wrapped in a `Place` struct with `added_at` set to the
migration time.

### Initial release

Core CLI tool with `add`, `list`, `go`, `rm` commands. Config stored as JSON at
`~/.config/places/places.json` with auto-created directory. Shell hook uses
marker-based injection (`# BEGIN/END places shell-hook`) for clean
install/uninstall. Follows patterns from the `notify` CLI tool: manual arg
parsing, `fatal()` helper, stdlib only.
