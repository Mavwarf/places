# TODO

## Bugs / Missing

- [x] ~~**cmd.exe support**~~ — `places shell-hook install --shell cmd` creates `p.bat` next to `places.exe` *(Mar 1)*
- [x] ~~**Windows PowerShell hook install**~~ — `places shell-hook install` now auto-detects PowerShell; use `--shell powershell` to target explicitly *(Feb 28)*

## Features

- [ ] **Interactive select with arrow keys** — replace numbered input with cursor up/down navigation and Enter to confirm; requires raw terminal input (consider `golang.org/x/term` or manual ANSI escape handling)
- [ ] **Desktop app** (`places app`) — Windows GUI showing all saved places; each place has a "PowerShell" and "cmd" button that opens a terminal window at that directory (similar to `notify-app` using Wails)
- [ ] **Fuzzy matching** — `p not` matches `notify`; no need for exact names
- [ ] **Auto-name on add** — `places add` without a name derives it from the directory (e.g. `/cli_tools/notify` → `notify`)
- [ ] **Tags/groups** — `p add notify --tag work`, then `p list --tag work` to filter
- [ ] **Import/export** — `places export > places.json` / `places import < places.json` for syncing across machines
- [ ] **Spawn shell** (`places cd <name>`) — open a new shell in that directory (no hook needed, works everywhere)

## Improvements

- [ ] Sort select menu by most recently used (instead of alphabetical)
- [ ] `places rename <old> <new>` command
- [ ] Tab completion for place names (bash/zsh/PowerShell)
- [ ] Validate that saved paths still exist on `list` / `select` (warn if missing)
- [ ] Color output — highlight place names, dim paths, color stats
- [ ] `places stats` — global usage summary (total uses, most used place, least used)
- [ ] `places edit` — open `places.json` in `$EDITOR`
