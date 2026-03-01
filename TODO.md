# TODO

## Bugs / Missing

- [x] ~~**cmd.exe support**~~ — `places shell-hook install --shell cmd` creates `p.bat` next to `places.exe` *(Mar 1)*
- [x] ~~**Windows PowerShell hook install**~~ — `places shell-hook install` now auto-detects PowerShell; use `--shell powershell` to target explicitly *(Feb 28)*
- [x] ~~**Relative path resolution**~~ — `places add name .` now resolves to absolute path *(Mar 1)*
- [x] ~~**Path separator normalization**~~ — mixed `/` and `\` normalized to OS-native on load *(Mar 1)*

## Features

- [x] ~~**Interactive select with arrow keys**~~ — cursor up/down navigation with Enter to confirm, Esc to cancel; raw terminal input via Windows Console API / Unix termios *(Mar 1)*
- [x] ~~**Desktop app** (`places app`)~~ — Wails v2 GUI showing all saved places with action buttons (PowerShell, Claude, cmd, Explorer); sorting by name/usage/last used/added; add/remove places from GUI *(Mar 1)*
- [x] ~~**`p app` command**~~ — launches `places-app.exe` from the CLI, added to all shell hook passthrough lists *(Mar 1)*
- [x] ~~**Sorting in desktop app**~~ — sort by name, most used, last used, date added *(Mar 1)*
- [x] ~~**Claude button**~~ — opens PowerShell at directory and starts `claude` CLI *(Mar 1)*
- [x] ~~**"places dashboard" title**~~ — updated page title, heading, and Wails window title *(Mar 1)*
- [x] ~~**App icon**~~ — blue circle with white "P" (Tokyo Night accent), embedded via go-winres *(Mar 1)*
- [x] ~~**GitHub wiki**~~ — getting started, use cases, CLI reference, desktop app, shell hooks *(Mar 1)*
- [x] ~~**GitHub Actions**~~ — CI (vet + build check) and release workflow (v* tags, multi-platform binaries) *(Mar 1)*
- [x] ~~**Fuzzy matching**~~ — `p not` matches `notify`; substring match with single-result disambiguation *(Mar 1)*
- [x] ~~**Auto-name on add**~~ — `places add` without a name derives it from the directory (e.g. `/cli_tools/notify` → `notify`) *(Mar 1)*
- [x] ~~**System tray**~~ — tray icon with right-click menu for quick access to places, double-click to reopen dashboard; hide-on-close keeps app running *(Mar 1)*
- [x] ~~**Spawn shell** (`p shell <name>`)~~ — open a new terminal at that directory (no hook needed) *(Mar 1)*
- [x] ~~**Prune** (`p prune`)~~ — bulk-remove all places where the directory no longer exists *(Mar 1)*
- [x] ~~**Reverse lookup** (`p where`)~~ — if cwd is a bookmarked directory, print its name *(Mar 1)*
- [x] ~~**Open in editor** (`p code <name>`)~~ — open directory in VS Code *(Mar 1)*
- [x] ~~**Auto-start on login** (`p autostart [on|off]`)~~ — registry-based autostart for Windows tray app *(Mar 1)*
- [x] ~~**Left-click tray opens menu**~~ — left-click now shows the menu, double-click opens dashboard *(Mar 1)*
- [x] ~~**Color schemes**~~ — 6 themes (dark, light, nord, dracula, solarized, gruvbox) with toggle button and localStorage persistence *(Mar 1)*
- [x] ~~**Browse button in add form**~~ — "…" button next to the path input opens a native folder picker dialog via Wails `OpenDirectoryDialog` *(Mar 1)*
- [x] ~~**Tags/groups** — `p add notify --tag work`, then `p list --tag work` to filter~~ *(Mar 1)*
- [ ] **Import/export** — `places export > places.json` / `places import < places.json` for syncing across machines
- [ ] **Temporary places** (`p add --temp`) — auto-expire after N days or on next prune
- [ ] **Notes** (`p add api --note "billing REST API"`) — attach a description, shown in list and desktop app
- [ ] **Clone + bookmark** (`p clone <git-url> [name]`) — git clone into a workspace dir and auto-add as a place
- [ ] **Global hotkey** — system-wide keyboard shortcut to open the tray menu or dashboard

## Script-Friendly

- [x] ~~**`p list --json`**~~ — machine-readable output for scripting / integrations *(Mar 1)*
- [x] ~~**`p exists <name>`**~~ — exit code 0/1 for use in shell scripts *(Mar 1)*
- [ ] Tab completion for place names (bash/zsh/PowerShell)

## Tech Debt

- [x] ~~**`config.Save()` errors ignored**~~ — added error checks in `cmdGo()` and `cmdSelect()` *(Mar 1)*
- [x] ~~**Missing path validation in `handleAdd()`**~~ — desktop app now checks path exists and is a directory *(Mar 1)*
- [x] ~~**Unquoted path in cmd launch**~~ — `app.go` and `tray.go` now quote paths in `cd /d` *(Mar 1)*
- [x] ~~**No HTTP client timeout in `waitForServer()`**~~ — added 200ms client timeout *(Mar 1)*
- [x] ~~**Inconsistent error formatting**~~ — `shellhook.go` now uses `fatal()` helper *(Mar 1)*
- [x] ~~**Silent tray menu failure**~~ — `addPlaceMenus()` now shows disabled error item in tray *(Mar 1)*
- [x] ~~**Terminal launch commands duplicated 3x**~~ — extracted to `internal/launcher` package *(Mar 1)*
- [x] ~~**Sorting logic duplicated**~~ — moved `SortedNames()` to `internal/config`, removed duplicates *(Mar 1)*
- [x] ~~**`fuzzyFind()` doesn't distinguish "not found" from "ambiguous"**~~ — now lists matching names on ambiguous query *(Mar 1)*
- [x] ~~**Hardcoded port 8822**~~ — now configurable via `PLACES_PORT` env var or `--port` flag *(Mar 1)*
- [ ] **Duplicate `jsonPlace` struct** — defined in both `commands.go` and `internal/app/app.go` (accepted: different fields needed)

## Improvements

- [x] ~~Sort select menu by most recently used (instead of alphabetical)~~ *(Mar 1)*
- [x] ~~`places rename <old> <new>` command~~ (alias: `mv`) *(Mar 1)*
- [x] ~~Validate that saved paths still exist on `list` / `select` (warn if missing)~~ *(Mar 1)*
- [x] ~~Color output~~ — names in green, paths in cyan, stats dimmed, warnings in yellow *(Mar 1)*
- [x] ~~`places stats` — global usage summary (total uses, most used place, least used)~~ *(Mar 1)*
- [x] ~~`places edit`~~ — open `places.json` in `$EDITOR` or specified editor *(Mar 1)*
- [x] ~~`places init`~~ — one-command setup: installs shell hooks for detected shell + cmd on Windows *(Mar 1)*
