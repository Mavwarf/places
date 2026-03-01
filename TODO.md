# TODO

## Bugs / Missing

- [ ] **cmd.exe support** — no shell hook for Windows cmd terminal; needs a `p.bat` wrapper or doskey macro to capture `places go` output and `cd` to it
- [ ] **Windows PowerShell hook install** — `places shell-hook install` only writes to the pwsh (PowerShell Core) profile; Windows PowerShell (`powershell.exe`) has a separate profile path that must be installed manually

## Improvements

- [ ] Sort select menu by most recently used (instead of alphabetical)
- [ ] `places rename <old> <new>` command
- [ ] Tab completion for place names (bash/zsh/PowerShell)
- [ ] Validate that saved paths still exist on `list` / `select` (warn if missing)
