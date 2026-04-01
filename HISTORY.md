# History

## Features

- Session history view — "Sessions" toggle in header shows today/this week timeline with colored bars per place, grouped sessions per row, summary cards; hides places UI (sort/filter/recent/compact) when active *(Mar 30)*
- Dashboard UX — visible column drag handles in place rows; git dirty tooltip shows changed files; recent bar docked to bottom above footer; header buttons reordered (Sessions, + left; Recent, Compact, 🎨, ⚙ right); merged nearby sessions in timeline; bigger fav/health icons; "Confirm fresh session" pref toggle; custom input/confirm dialogs; tag overflow menu with AND filter; always-visible Clear button *(Mar 30)*
- Aggregate git status for multi-repo places — places containing sub-repos show "2 repos: 1 clean, 1 dirty" with per-repo tooltip; health dots and dirty filter work for both *(Apr 1)*
- Tech debt cleanup — SQLite error logging in session tracker, strings.Cut for key parsing, validate default hidden/actions lists, shutdown hook closes tracker, single-instance check timeout *(Mar 31)*
- UI polish — Sessions button centered in header; screenshot badge moved right and clickable to exit; session tooltips show action breakdown; keyboard shortcuts F2 (screenshot) and F3 (sessions); fixed search-as-you-type conflict with screenshot key *(Mar 31)*
- 37 color themes — added Solarized Light, One Light, Quiet Light, Alabaster, Material, Zenburn; themes sorted by background darkness within Dark/Warm/Light groups *(Mar 31)*
- 31 color themes — added Tokyo Night, Ayu Dark/Light, Synthwave, Poimandres, Vesper, Palenight, Flexoki Dark/Light, High Contrast Dark/Light, GitHub Dark/Light, Night Owl, Cobalt2, Horizon; multi-column picker (🎨) grouped by Dark/Warm/Light with live preview *(Mar 29)*
- macOS support — full desktop app port: OS-aware action buttons (📁 Finder / >_ Terminal instead of dir/PS/cmd), iTerm2 auto-detection with Terminal.app fallback (preference toggle), running session detection via `ps`+`lsof`, Cmd+click for YOLO mode, hidden Windows-only UI (virtual desktops, systray); Makefile (`make mac`) for build+install+`.app` bundle; CI/release workflows with `.app` bundle zips *(Mar 29)*
- Session time tracking — SQLite database records session start/end times; tooltip shows elapsed time and today's total; graceful shutdown and crash recovery via last_seen_at polling *(Mar 28)*
- Running session indicator — detects active sessions for all action types (Claude, VS Code, PowerShell, cmd, Explorer, custom actions) via process and window title scanning; green glow on action buttons, recent chips, and fav chips; tooltip shows desktop number and elapsed time; opt-in via preferences *(Mar 27)*
- Claude effort level — configurable per-place and global default (low/medium/high/max) via ⋯ menu and preferences; passed as `--effort` flag on launch *(Mar 27)*
- Preferences reorganization — grouped into Claude, Actions, and Integration sections with clear labels *(Mar 27)*
- Executable picker for custom actions — browse button in action manager opens native file picker, auto-fills command template and name/label *(Mar 27)*
- Pin to all desktops — globe button in header pins the dashboard to all virtual desktops via VirtualDesktopAccessor.dll *(Mar 27)*
- Place health icons — accessible shape+color indicators: ✓ green (clean git), ◆ yellow (dirty), ✗ red (missing dir); hidden for non-git places *(Mar 27)*
- Favorites skip recent — launching a favorited action no longer adds duplicates to the recent list *(Mar 27)*
- Favorite action reordering — drag to reorder pinned favorite actions *(Mar 27)*
- Favorite actions — Ctrl+click a recent chip to pin it as a favorite; favorites shown with ★ icon to the right of recents; right-click to remove; synced to server and shown in system tray *(Mar 27)*
- Action manager editing — edit button per custom action populates the form for modification; custom actions can be auto-assigned to new places via defaults section *(Mar 26)*
- Configurable Claude shell — preference to launch Claude in cmd (default) or PowerShell, via gear menu; "Set tab title" toggle controls whether Claude can update the terminal tab title with status indicators *(Mar 26)*
- Action manager — define, delete, and overwrite custom actions from the dashboard via gear menu; default action visibility for new places *(Mar 26)*
- Notify hook setup — per-place "Setup notify hooks" in place menu auto-creates .claude/settings.json with Stop/Notification hooks; notify.exe path configurable in preferences *(Mar 26)*
- Tray recent menu — recent actions synced to server and shown in system tray under "Recent" submenu; all places grouped under "Places" submenu *(Mar 26)*
- Compact mode — "Compact" toggle in header switches to a dense single-line view showing name, tags, stats, and action buttons; hides path, notes, git status *(Mar 26)*
- Search-as-you-type — typing anywhere in the dashboard auto-focuses the filter input *(Mar 26)*
- Favorites pin to top — "Pin ★" toggle in filter bar floats favorites above non-favorites in all sort modes (except desktop sort); default on, persisted *(Mar 26)*
- Stale place warning — places with missing directories show a red left border and reduced opacity *(Mar 26)*
- 15 color themes — added Catppuccin, Catppuccin Latte, One Dark, Rose Pine, Rose Pine Dawn, Kanagawa, Everforest, Everforest Light, Monokai; theme picker dropdown with live preview on hover; right-click theme button to restore startup theme *(Mar 26)*
- Sort by desktop — new sort option groups places by virtual desktop assignment with section headers, unassigned at bottom *(Mar 26)*
- Recent action history — toggleable "Recent" bar in the header shows last 8 place+action launches as one-click chips; tracks Claude mode (YOLO/fresh); right-click to remove with toast confirmation; persisted in localStorage *(Mar 26)*
- Dashboard UX — filter/sort state persists across restarts; close button always quits (no more hide-to-tray); removed redundant minimize/quit buttons from header; reduced horizontal padding; dynamic virtual desktop count from DLL *(Mar 26)*
- Claude YOLO mode — Ctrl+click CL button launches Claude with `--dangerously-skip-permissions`; Shift+click (fresh session) now shows a confirmation dialog; tab title simplified to `claude <name>` *(Mar 24)*
- Dashboard polish — subtitle changed to "workspace navigator", empty state message when filters hide all places, place list top padding *(Mar 5)*
- Tech debt cleanup (22 items) — UTC build time, inline edit Escape fix, toast stacking, JS var→const/let, renderNote helper, dropdown CSS classes, gitStatus/mergeConfig extraction, geometry error logging, tray nil guard and save error check, cmdListJSON error check, README buildTime ldflags, ExpandAction doc comment *(Mar 5)*
- Build timestamp in dashboard footer — status bar shows "2026-03-05 09:30 UTC · v0.3.8" with build time injected via ldflags; consistent UTC format across CI and local builds *(Mar 5)*
- Tech debt cleanup (9 items) — CSS variable button colors, geometry save errors, appTitle const, modifyPlace helper, readKey removal, dropdown-sep class, deterministic stats, Serve callbacks struct, shared action allowlist *(Mar 4)*
- Sticky header with collapsible add form — header, sort bar, and filter bar stay fixed while places list scrolls; add form hidden behind a "+" toggle button; full-width header border matches footer *(Mar 4)*
- Text filter — search input in the filter bar filters places by name, path, or note (case-insensitive); combines with tag/fav/dirty filters *(Mar 3)*
- Tag exclusion filter — right-click a tag chip to exclude places with that tag (red + strikethrough); left-click still includes; a tag can only be in one state *(Mar 3)*
- Wiki link in status bar — added link to the Desktop App wiki page alongside the existing GitHub link *(Mar 3)*
- Auto git fetch on startup — git status is fetched for all places automatically when the dashboard opens; no manual clicking needed *(Mar 3)*
- Git dirty filter — new "Git dirty" chip in the filter bar shows only places with uncommitted changes *(Mar 3)*
- Status bar — fixed bar at the bottom of the window showing author/GitHub links (left), place count with filter ratio (center), and build version (right); always visible regardless of scroll *(Mar 3)*
- Version display — build version injected via `-ldflags` at compile time; shown in the status bar *(Mar 3)*
- Dashboard filter rework — tags are now toggleable multi-select chips (OR logic); ★ favorite chip moved from sort bar into the unified filter bar alongside tags; "Clear" button appears when any filter is active and resets everything; "All" button removed *(Mar 3)*
- Git status UX — unchecked places show a dim "no status" hint; bulk "Update git" only fetches for currently visible (filtered) places; git subprocesses run hidden on Windows (no flashing cmd windows) *(Mar 3)*
- Per-place action hiding — right-click built-in action buttons (PS, cmd, Claude, Code, Explorer) to hide them for a specific place; toggle visibility via the place menu; persisted in `hidden_defaults` field *(Mar 3)*
- Auto-refresh popup fix — dashboard auto-refresh now skips reload while action assign dropdown or desktop select is open, preventing popup destruction mid-interaction *(Mar 3)*
- Notes — attach text descriptions to places (`p note <name> [text]`); shown as inline subtitles in the dashboard; click to edit, hover to add; included in JSON output and import/export *(Mar 2)*
- Import/Export — `p export` dumps all places and actions as JSON; `p import <file>` merges from file (skip existing); dashboard has Export/Import buttons for one-click backup *(Mar 2)*
- Custom actions — define global actions with shell command templates, assign them to individual places; buttons appear in the dashboard and system tray alongside built-in actions; right-click a custom action button to unassign it from that place; `{path}` and `{name}` placeholders expanded at runtime; Windows uses `SysProcAttr.CmdLine` to bypass Go's quote escaping for `cmd /c` *(Mar 2)*
- Code comments — added documentation across all 16 source files covering Windows APIs, ANSI escapes, platform-specific patterns, concurrency model, and architectural decisions *(Mar 2)*
- Drag-and-drop path — drag a folder from Explorer onto the dashboard to fill in the add form's path input *(Mar 2)*
- Always on top — pin button in the dashboard header toggles the window to stay above all other windows; state persists across restarts via localStorage *(Mar 2)*
- Claude tab title — launching Claude from the dashboard or tray sets the terminal tab title to "claude \<name\>"; uses Windows Terminal `--suppressApplicationTitle` to prevent override *(Mar 2)*
- Place name validation — names restricted to letters, numbers, hyphens, underscores, dots (max 64 chars); enforced in CLI and desktop app *(Mar 2)*
- Stable dashboard sorting — places with equal sort keys (e.g. never-used places sorted by last used) now use alphabetical name as tiebreaker; no more shuffling on auto-refresh *(Mar 2)*
- External links open in system browser — footer links in the dashboard now open in the default browser instead of navigating inside the WebView *(Mar 2)*
- Global hotkey (Win+Alt+P) — system-wide shortcut to open the dashboard; switches to the dashboard's virtual desktop if on a different one *(Mar 2)*
- Desktop switch button — → button next to the virtual desktop dropdown jumps to that desktop without launching a tool *(Mar 2)*
- Virtual desktop (`places desktop <name> <0-4>`) — assign a Windows virtual desktop to a place; dashboard and tray switch to that desktop before launching tools; uses VirtualDesktopAccessor.dll *(Mar 2)*
- Auto-refresh in desktop app — polls /api/places every 3 seconds so CLI-added places appear automatically *(Mar 2)*
- Usage tracking from app/tray — clicking action buttons (PS, cl, VS, >_, dir) now increments use count and updates last-used timestamp; UI refreshes immediately *(Mar 2)*
- Desktop app (`places app`) — Wails v2 native GUI with action buttons (PowerShell, Claude, cmd, VS Code, Explorer), sorted place list, add/remove places from the dashboard; HTTP server + WebView redirect architecture *(Mar 1)*
- System tray — tray icon with right-click menu for quick access to places, double-click to reopen dashboard; hide-on-close keeps the app running in the background *(Mar 1)*
- Tags (`places tag`/`untag`/`tags`) — organize places with tags; `p add --tag work`, `p list --tag work`; tag badges and filter bar in the desktop app *(Mar 1)*
- Color themes — 6 themes (Dark, Light, Nord, Dracula, Solarized, Gruvbox) in the desktop app; toggle button with localStorage persistence *(Mar 1)*
- Screenshot mode — press S in the desktop app to anonymize place names; work-tagged places also get paths anonymized; mappings persist in localStorage *(Mar 1)*
- Window controls — minimize and quit buttons in the dashboard header; pin button for always-on-top *(Mar 1)*
- Browse button in add form — "…" button opens a native folder picker dialog via Wails `OpenDirectoryDialog` *(Mar 1)*
- Open in VS Code (`places code <name>`) — open a place's directory in VS Code *(Mar 1)*
- Open terminal (`places shell <name>`) — open a new terminal at a place's directory without shell hook *(Mar 1)*
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

## 2026-03-05

### Build timestamp

The dashboard status bar now shows the build date and time next to the version
tag (e.g. "2026-03-05 09:30 UTC · v0.3.8"). Build time is injected via
`-ldflags "-X 'main.buildTime=...'"` at compile time. Both CI (`release.yml`)
and local builds (`CLAUDE.md`, `SKILL.md`) use UTC with a " UTC" suffix for
consistency.

### Tech debt cleanup (22 items)

Comprehensive review-driven cleanup across CLI, desktop app, and dashboard:

- **Inline edit Escape fix** — pressing Escape in name/path/note edit now removes
  the blur listener before finishing, preventing a spurious `save()` call
- **Toast stacking** — multiple simultaneous toasts offset vertically (48px each)
  instead of overlapping at the same position
- **JS modernization** — all `var` declarations (global and local) converted to
  `const`/`let` throughout the dashboard; zero `var` remaining
- **renderNote() helper** — extracted from inline IIFE in place card template
- **CSS cleanup** — dropdown item colors via classes (`.dropdown-item-custom`,
  `.dropdown-item-rm`) instead of inline `style.color`; file input uses `.hidden`
  class; `.btn-rm` gets `position: relative` in CSS instead of JS
- **HTTP handler extraction** — `gitStatus()` helper from `handleGitStatus`,
  `mergeConfig()` helper from `handleImport`
- **Error handling** — geometry save/load logs marshal/unmarshal errors to stderr;
  tray `recordTrayUse` logs save failures; `cmdListJSON` checks encode error
- **Defensive coding** — nil place guard in tray menu loop
- **Build consistency** — UTC build time in CI, local, and SKILL.md; README
  build command includes `buildTime` ldflags
- **Code cleanup** — removed unused `version`/`buildTime` vars from CLI, stale
  `buildDate` comment from places-app, simplified `cmdEscape` to double-quoting,
  renamed `{{build_date}}` placeholder to `{{build_time}}`
- **Documentation** — `ExpandAction()` doc comment documents quoting requirement

---

## 2026-03-04

### Sticky header

The dashboard header section (title bar, add form, sort bar, filter bar) is now
fixed at the top of the window. Only the places list scrolls. Uses CSS flexbox
with `overflow-y: auto` on the place list container. A full-width border at the
bottom of the header matches the status bar border at the bottom.

### Collapsible add form

The add-new-place form is now hidden by default behind a **+** button in the
header row (left of the theme toggle). Clicking it expands the form; the form
auto-collapses after a successful add. Reclaims vertical space for the place
list in the common case.

### Tighter header spacing

Reduced vertical gaps throughout the header section: header margin 20→10px,
sort bar margin 12→6px, tag bar margin 12→0, header padding 12→8px, add form
margin 24→10px.

### Tech debt cleanup (9 items)

- Added `--claude`/`--code` CSS variables to all 6 theme blocks (were hard-coded hex)
- `saveGeometry()` logs to stderr on write failure (was silently ignored)
- Extracted `appTitle` constant replacing 3 hard-coded `"places dashboard"` strings
- Extracted `modifyPlace()` helper for load-modify-save boilerplate in 5 commands
- Removed redundant `readKey()` wrapper, calling `readKeyCode()` directly
- Added `.dropdown-sep` CSS class replacing inline `style.cssText` on separator divs
- `cmdStats` iterates `SortedNames()` for deterministic most/least used output
- `Serve()` replaced 7 callback params with `Callbacks` struct
- Shared `defaultActions` map for `handleOpen` and `handleToggleDefault` allowlists

---

## 2026-03-03

### Text filter

Search input in the filter bar (after the Git dirty chip) filters places by
name, path, or note — case-insensitive substring match. The input is a
persistent DOM element that survives filter bar re-renders, so typing is never
interrupted. Combines with all other filters (tags, favorites, git dirty) via
AND. Clear resets the text input along with all other filters.

### Tag exclusion filter

Right-click a tag chip to exclude places with that tag instead of including
them. Excluded tags appear in red with strikethrough. A tag can be in one of
three states: inactive, included (left-click, accent color), or excluded
(right-click, red). Switching between states is automatic — left-clicking an
excluded tag switches it to included, and vice versa.

### Wiki link in status bar

Added a "Wiki" link to the dashboard status bar (bottom left), pointing to the
Desktop App wiki page. Appears alongside the existing author and GitHub links.

### Auto git fetch on startup

Git status is now fetched automatically for all places when the dashboard opens,
running in parallel via `Promise.all`. Badges populate silently without a toast.
The manual "Update git" button and per-place **git** button remain for on-demand
refresh.

### Git dirty filter

New **Git dirty** chip in the filter bar (between the tag chips and the Clear
button). When active, shows only places where git status has been fetched and
the working directory has uncommitted changes. Combines with tag and favorite
filters.

### Status bar

The author/GitHub links moved from a scrollable div below the header to a fixed
status bar at the bottom of the window. Three sections:

- **Left** — author name + GitHub link
- **Center** — place count; shows "12 places" when unfiltered, "5 / 12 places" when filtered
- **Right** — build version (e.g. "v0.3.3")

Always visible regardless of scroll position. Toasts repositioned above the bar.

### Version display

Build version is injected at compile time via `-ldflags "-X main.version=..."`.
The `app.Version` package variable is replaced into a `{{version}}` placeholder
in the served HTML. Shows "dev" when built without ldflags.

### Per-place action hiding

Right-click any built-in action button (PS, cmd, Claude, Code, Explorer) on a
place to hide it. Hidden actions are tracked in the `hidden_defaults` field of
each place. The place menu ("...") shows all default actions with checkmarks to
toggle visibility. State persists in `places.json` and survives restarts. The
`/api/toggle-default` endpoint handles the toggle; a shared `defaultActions` map
validates action names.

### Security: URL open fix

Replaced `exec.Command("cmd", "/c", "start", "", url)` with
`exec.Command("rundll32", "url.dll,FileProtocolHandler", url)` in
`handleOpenURL`. The old approach passed URLs through cmd.exe's parser, where
`&` in query strings could be interpreted as command separators. The new
approach avoids cmd.exe entirely.

### Compact place rows

Reduced vertical padding in place rows from `10px 14px` to `5px 14px 7px` for
a more compact layout.

### Dead code cleanup

Removed unused `showActionDropdown()` function (31 lines) from the frontend —
superseded by the place menu (`showPlaceMenu`).

---

## 2026-03-02

### Custom actions

Define reusable shell command templates and assign them to specific places.
Custom action buttons appear alongside the built-in PS/cl/VS/>_/dir buttons —
only on places they're assigned to.

- CLI: `places action add/rm/list/assign/unassign` subcommands
- Config: `Action` struct with `label` and `cmd` fields; `Place.Actions` list
- Templates: `{path}` expands to the place directory, `{name}` to the place name
- Dashboard: custom action buttons per place (left of built-in buttons),
  "+" dropdown to assign actions, right-click a custom action button to
  unassign it from that place
- System tray: custom action submenu items per place
- Execution: platform-specific `launcher.Shell()` — Windows uses
  `SysProcAttr.CmdLine` to pass commands raw to `cmd /c`, bypassing Go's
  argument escaping that breaks embedded quotes; Unix uses `sh -c`
- GUI apps on Windows need `start ""` prefix to launch from the background
- PowerShell mangles embedded quotes in `--cmd` values — use `p edit` to
  define actions with complex commands directly in `places.json`

Example (`places.json`):
```json
{
  "rider": { "label": "JR", "cmd": "start \"\" \"C:\\Program Files\\JetBrains\\Rider\\bin\\rider64.exe\" \"{path}\"" },
  "godot": { "label": "GD", "cmd": "start \"\" \"path\\to\\godot.exe\" -e --path \"{path}\"" }
}
```

### Code comments

Added comments across all 16 Go source files so a new contributor can understand
the codebase without prior context. Focus areas:

- **Windows APIs** — `ReadConsoleInputW` vs `os.Stdin.Read`, `SetWindowPos` z-order
  constants, `RegisterHotKey` + `GetMessageW` threading, `DETACH_PROCESS` flag,
  `FreeConsole`, ICO file format for `SetIcon`
- **Unix terminal** — termios `ICANON`/`ECHO` flags, VT100 escape sequence parsing
- **ANSI escapes** — cursor hide/show (`DECTCEM`), line clear, color codes in selector
- **Concurrency** — `config.Lock/Unlock` for read-modify-write cycles, manual unlock
  in `handleOpen` to avoid holding the lock during process launch, `runtime.LockOSThread`
  for Windows message loops
- **Architecture** — shell hook marker strategy, cmd.exe temp file workaround,
  Wails HTTP redirect + `BindingsAllowedOrigins`, `originGuard` CSRF protection,
  atomic config save via temp+rename, VirtualDesktopAccessor 0/1 indexing

### Drag-and-drop path

Drag a folder from Windows Explorer onto the dashboard window to fill in the add
form's path input. Uses the WebView2 native `postMessageWithAdditionalObjects`
API to resolve full file paths from the drop event, bypassing browser security
restrictions that hide paths from JavaScript.

- `BindingsAllowedOrigins` allowlists the HTTP server origin for WebView2 IPC
- Go `OnFileDrop` callback stores the resolved path, exposed via `/api/last-drop`
- JS polls the endpoint after each drop to retrieve the path

### Always on top

The dashboard header now has a **pin** button (📌) between the theme toggle and
the minimize button. Clicking it toggles always-on-top mode — the window stays
above all other windows. Clicking again reverts to normal behavior. The state is
saved in localStorage, so if you pin the window and restart the app, it
re-applies automatically on load.

- Windows: `SetWindowPos` with `HWND_TOPMOST` / `HWND_NOTOPMOST`
- Non-Windows: no-op stub (feature is Windows-only)
- New `/api/topmost` POST endpoint accepts `{"on_top": true/false}`
- Pin button uses accent color when active

### Claude tab title

Launching Claude Code from the dashboard or system tray now sets the terminal
tab title to **"Claude Code - \<name\>"** (e.g. "Claude Code - notify"). This
makes it easy to identify which Claude session belongs to which project when
running multiple sessions.

Uses Windows Terminal's `wt new-tab --suppressApplicationTitle` to prevent
Claude from overriding the title. Falls back to `cmd /c start` if Windows
Terminal is not available (title may be overridden in that case).

### Tech debt: name validation, error sanitization, dedup, ICO errors

Four tech debt fixes in one pass:

- **Place name validation** — new `config.ValidateName()` rejects names with
  special characters that could break shell hooks or HTML rendering. Valid names
  use letters, numbers, hyphens, underscores, dots (max 64 chars, must not start
  with a dash). Enforced in `cmdAdd`, `cmdRename`, and `handleAdd`.
- **HTTP error sanitization** — all HTTP error responses from config Load/Save
  now return generic messages ("failed to load config", "failed to save config")
  instead of raw `err.Error()` which could leak filesystem paths.
- **Duplicate filter logic** — extracted `config.FilterNames()` shared helper,
  replacing duplicate tag/favorite filtering in `cmdList()` and `cmdListJSON()`.
- **pngToICO error handling** — `pngToICO()` now returns `([]byte, error)`;
  caller falls back to raw PNG bytes if ICO conversion fails.

### Stable dashboard sorting

Sorting in the dashboard now uses the place name as a tiebreaker when the
primary sort key is equal. Previously, places with the same value (e.g.
never-used places sorted by "last used") would shuffle randomly on each
3-second auto-refresh. Affects "most used", "last used", and "added" sort modes.

### External links open in system browser

Footer links (author website, GitHub repo) in the dashboard now open in the
system default browser instead of navigating inside the Wails WebView. A global
click interceptor catches all `<a target="_blank">` clicks and routes them
through a new `/api/open-url` endpoint, which validates the URL (https only)
and launches the platform browser (`cmd /c start` on Windows, `open` on macOS,
`xdg-open` on Linux).

### Global hotkey

Press **Win+Alt+P** from anywhere to open the places dashboard. If the
dashboard is on a different virtual desktop, you are switched to that desktop
first, then the window is shown.

- Registers `Win+Alt+P` via `RegisterHotKey` on a dedicated OS thread
- Finds the Wails window by title, queries its desktop via
  `GetWindowDesktopNumber`, switches with `GoToDesktopNumber`
- Graceful degradation: if the DLL is missing, the window still shows (no
  desktop switch); if the hotkey is already registered, the app runs without it

### Desktop switch button

Each place row in the desktop app now shows a **→** button next to the virtual
desktop dropdown (D1–D4). Clicking it switches to that desktop without
launching any tool. The button only appears when a desktop is assigned.

- New `/api/switch-desktop` endpoint calls `launcher.SwitchDesktop(n)`
- Button hidden by default, shown via CSS `.visible` class when `p.desktop > 0`

### App icon recolor

Changed the app icon background from blue (Tokyo Night accent) to orange,
matching the notify app icon. Same white "P" on orange circle.

### `p list` indentation fix

Fixed misaligned output when mixing favorite and non-favorite places. Non-
favorite entries now pad with two spaces to match the width of the ★ marker.

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

### Notes

Attach text descriptions to places via `p note <name> [text...]`. Omit text to
print the current note, use `--rm` to clear it. Notes are included in
`p list --json` output and preserved in import/export.

In the desktop app, notes appear as subtitles below the path — truncated with
ellipsis when too long. Click a note to edit it inline (Enter saves, Escape
cancels). Hover over a place without a note to see an "add note" prompt. Notes
are also anonymized in screenshot mode for work-tagged places.

### Import/Export

`p export` dumps the full config (places + actions) as JSON to stdout.
`p import <file>` merges from a file — adds new places and actions, skips
existing ones. The format matches `places.json`, making it round-trippable.

The desktop app has **Export** and **Import** buttons in the sort bar. Export
downloads a `places-export.json` file. Import opens a file picker, merges the
selected file, and shows a toast with counts (e.g. "3 places added, 2 skipped").

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

### Desktop app

Wails v2 native GUI launched via `p app`. Shows all saved places with action
buttons: **PS** (PowerShell), **cl** (Claude Code), **>_** (cmd), **VS** (VS
Code), **dir** (Explorer). Sort bar with Name, Most used, Last used, Added
sort modes. Add form with name + path inputs. Remove places via the UI.

Architecture: Go HTTP server on `127.0.0.1:8822` serving vanilla HTML/CSS/JS;
Wails WebView redirects to the HTTP server URL on startup. Shared config with
the CLI — changes sync immediately.

### Tags

Tag-based organization for places. CLI commands: `p tag <name> <tag>`,
`p untag <name> <tag>`, `p tags` (list all with counts). Tags can also be set
on creation: `p add api --tag work --tag backend`. Filter with `p list --tag work`.

Desktop app: tag badges on each place, **+** button to add via prompt, **×** to
remove, filter bar with toggleable tag chips.

### System tray

Desktop app runs in the system tray via `energye/systray`. Closing the window
hides to tray instead of exiting. Double-click to reopen. Right-click (or
left-click) for context menu: Open Dashboard, per-place submenus with PS/Claude/
cmd/Explorer actions, Refresh, Quit.

### Color themes

Six color schemes in the desktop app: Dark (Tokyo Night, default), Light, Nord,
Dracula, Solarized, Gruvbox. Toggle button in the header cycles through them.
Selection saved in localStorage. All colors use CSS variables.

### Screenshot mode

Press **S** to toggle — anonymizes place names with fantasy names. Work-tagged
places also get paths anonymized (preserving `C:\dev\repos\` prefix). Mappings
persist in localStorage for consistency across reloads.

### Window controls

Minimize and quit buttons in the dashboard header. Close hides to tray;
Shift+close fully exits. Window geometry (position + size) saved to
`~/.config/places/window.json` and restored on launch.

### Browse button

"…" button in the add form opens a native folder picker dialog (Wails
`OpenDirectoryDialog`). Selected directory fills the path input.

### Favorites

`p fav <name>` / `p unfav <name>` toggle favorite status. `p list --fav` and
`p list --json --fav` filter to favorites. `p list` shows ★ marker for favorites.

Desktop app: clickable star per place, ★ filter button in the sort bar.

### Open in VS Code and terminal

`p code <name>` opens a directory in VS Code. `p shell <name>` opens a new
terminal at that directory (no shell hook needed). Both support fuzzy matching.

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
