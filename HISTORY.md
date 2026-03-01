# History

## Features

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

## 2026-03-01

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
