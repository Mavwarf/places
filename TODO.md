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
- [x] ~~**Screenshot mode**~~ — press S in desktop app to anonymize names and work-tagged paths for screenshots *(Mar 1)*
- [x] ~~**Window controls**~~ — minimize and quit buttons in desktop app header *(Mar 1)*
- [x] ~~**Favorites** (`p fav`/`p unfav`)~~ — mark places as favorites, filter with `--fav` in list, star toggle and filter button in desktop app *(Mar 1)*
- [x] ~~**Virtual desktop** (`p desktop <name> <0-4>`)~~ — assign places to Windows virtual desktops; dashboard and tray switch desktop before launching tools; uses `VirtualDesktopAccessor.dll` *(Mar 2)*
- [x] ~~**Auto-refresh in desktop app**~~ — polls `/api/places` every 3 seconds so CLI-added places appear automatically *(Mar 2)*
- [x] ~~**Usage tracking from app/tray**~~ — action buttons (PS, cl, VS, >_, dir) now record use count and last-used timestamp *(Mar 2)*
- [x] ~~**Global hotkey** — Win+Alt+P opens the dashboard from anywhere, switching to its virtual desktop~~ *(Mar 2)*
- [x] ~~**Import/export** — `places export` / `places import <file>` for syncing across machines; dashboard Export/Import buttons~~ *(Mar 2)*
- [x] ~~**Notes** (`p note <name> [text]`) — attach a description, shown as subtitle in dashboard with inline editing~~ *(Mar 2)*
- [ ] **Type-to-filter in select** — start typing in the interactive picker to narrow results instead of just arrow keys
- [ ] **Git info in desktop app** — show current branch and dirty/clean status next to each place
- [x] ~~**Custom actions** — user-defined commands per place or globally, beyond the built-in PS/cmd/Claude/Explorer~~ *(Mar 2)*
- [x] ~~**Frecency sorting** — combine frequency + recency into a single score for smarter ordering in select and app~~ *(Mar 2)*
- [ ] **`p back`** — jump to the previous place you were at (like `cd -` but across sessions)
- [ ] **Inline editing in dashboard** — click a place's name or path to edit it inline; rename and path update via API
- [x] ~~**Always on top** — dashboard window stays above all other windows~~ *(Mar 2)*
- [x] ~~**Drag-and-drop path** — dragging a folder from Explorer onto the add form's path input fills in the path~~ *(Mar 2)*
- [x] ~~**Open links in default browser** — web links clicked in the dashboard (Wails WebView) now open in the system default browser via `/api/open-url` endpoint~~ *(Mar 2)*

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
- [x] ~~**Shell injection via path in launcher**~~ — single quotes in paths escaped with `''` for PowerShell `Set-Location` *(Mar 2)*
- [x] ~~**Unauthenticated HTTP API**~~ — `Origin` header validation rejects cross-origin requests from malicious websites *(Mar 2)*
- [x] ~~**Config file race condition**~~ — process-level mutex around Load→modify→Save cycles; atomic write via temp file + rename *(Mar 2)*
- [x] ~~**Missing `//go:build windows` on `shift_windows.go`**~~ — added explicit build tag *(Mar 2)*
- [x] ~~**Desktop number not validated in API**~~ — `handleDesktop` now rejects values outside 0–4 *(Mar 2)*
- [x] ~~**`handleAdd` doesn't resolve relative paths**~~ — HTTP API now calls `filepath.Abs` before storing *(Mar 2)*
- [x] ~~**`cmdAdd` silently overwrites existing places**~~ — CLI now warns on stderr when overwriting *(Mar 2)*
- [x] ~~**`beforeClose` reads `a.ctx` without `<-a.ready`**~~ — now waits on ready channel before accessing context *(Mar 2)*
- [x] ~~**`handlePlaces` returns random order**~~ — now uses `config.SortedNames` for stable alphabetical order *(Mar 2)*
- [x] ~~**Config migration ignores `Save` error**~~ — now checks and returns the error *(Mar 2)*
- [x] ~~**`Sscanf` for port parsing**~~ — replaced with `strconv.Atoi` in `places-app/main.go` *(Mar 2)*
- [ ] **Duplicate `jsonPlace` struct** — defined in both `commands.go` and `internal/app/app.go` (accepted: different fields needed)
- [ ] **`os.Exit(0)` bypasses cleanup** — `QuitApp()`, tray quit, and `beforeClose` skip deferred functions, Wails shutdown, HTTP graceful shutdown
- [ ] **Goroutine leak in `Detach`** — `go cmd.Wait()` goroutines accumulate for long-running child processes on non-Windows
- [x] ~~**Duplicate tag/fav filtering logic** — extracted `config.FilterNames()` shared helper, used by both `cmdList()` and `cmdListJSON()`~~ *(Mar 2)*
- [x] ~~**No place name validation** — `config.ValidateName()` rejects special chars; enforced in `cmdAdd`, `cmdRename`, and `handleAdd`~~ *(Mar 2)*
- [ ] **Unix escape key blocks in selector** — pressing Esc with no following bytes causes `readKeyCode` to block indefinitely (`term_unix.go`)
- [x] ~~**Ignored errors in `pngToICO`** — `pngToICO` now returns error; caller falls back to raw PNG on failure~~ *(Mar 2)*
- [ ] **`Cmd()` and `Claude()` launchers have no platform guard** — unconditionally build `cmd.exe` commands, fail on non-Windows
- [ ] **`termios` struct is Linux-specific** — ioctl numbers and struct layout in `term_unix.go` won't work on macOS/FreeBSD
- [x] ~~**Error responses leak internal paths** — HTTP handlers now return generic messages instead of raw `err.Error()`~~ *(Mar 2)*
- [x] ~~**Unstable sort in dashboard** — added alphabetical name tiebreaker to all sort modes so equal-key places stay stable across auto-refreshes~~ *(Mar 2)*

## Improvements

- [x] ~~Sort select menu by most recently used (instead of alphabetical)~~ *(Mar 1)*
- [x] ~~`places rename <old> <new>` command~~ (alias: `mv`) *(Mar 1)*
- [x] ~~Validate that saved paths still exist on `list` / `select` (warn if missing)~~ *(Mar 1)*
- [x] ~~Color output~~ — names in green, paths in cyan, stats dimmed, warnings in yellow *(Mar 1)*
- [x] ~~`places stats` — global usage summary (total uses, most used place, least used)~~ *(Mar 1)*
- [x] ~~`places edit`~~ — open `places.json` in `$EDITOR` or specified editor *(Mar 1)*
- [x] ~~`places init`~~ — one-command setup: installs shell hooks for detected shell + cmd on Windows *(Mar 1)*
