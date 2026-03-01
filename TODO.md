# TODO

## Bugs / Missing

- [x] ~~**cmd.exe support**~~ — `places shell-hook install --shell cmd` creates `p.bat` next to `places.exe` *(Mar 1)*
- [x] ~~**Windows PowerShell hook install**~~ — `places shell-hook install` now auto-detects PowerShell; use `--shell powershell` to target explicitly *(Feb 28)*
- [x] ~~**Relative path resolution**~~ — `places add name .` now resolves to absolute path *(Mar 1)*
- [x] ~~**Path separator normalization**~~ — mixed `/` and `\` normalized to OS-native on load *(Mar 1)*

## Features

- [ ] **Interactive select with arrow keys** — replace numbered input with cursor up/down navigation and Enter to confirm; requires raw terminal input (consider `golang.org/x/term` or manual ANSI escape handling)
- [ ] **Desktop app** (`places app`) — Windows GUI showing all saved places; each place has a "PowerShell" and "cmd" button that opens a terminal window at that directory (similar to `notify-app` using Wails)
- [ ] **Fuzzy matching** — `p not` matches `notify`; no need for exact names
- [x] ~~**Auto-name on add**~~ — `places add` without a name derives it from the directory (e.g. `/cli_tools/notify` → `notify`) *(Mar 1)*
- [ ] **Tags/groups** — `p add notify --tag work`, then `p list --tag work` to filter
- [ ] **Import/export** — `places export > places.json` / `places import < places.json` for syncing across machines
- [ ] **Spawn shell** (`places cd <name>`) — open a new shell in that directory (no hook needed, works everywhere)

## Improvements

- [x] ~~Sort select menu by most recently used (instead of alphabetical)~~ *(Mar 1)*
- [x] ~~`places rename <old> <new>` command~~ (alias: `mv`) *(Mar 1)*
- [ ] Tab completion for place names (bash/zsh/PowerShell)
- [x] ~~Validate that saved paths still exist on `list` / `select` (warn if missing)~~ *(Mar 1)*
- [ ] Color output — highlight place names, dim paths, color stats
- [x] ~~`places stats` — global usage summary (total uses, most used place, least used)~~ *(Mar 1)*
- [ ] `places edit` — open `places.json` in `$EDITOR`
