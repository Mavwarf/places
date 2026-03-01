# TODO

## Bugs / Missing

- [x] ~~**cmd.exe support**~~ — `places shell-hook install --shell cmd` creates `p.bat` next to `places.exe` *(Mar 1)*
- [x] ~~**Windows PowerShell hook install**~~ — `places shell-hook install` now auto-detects PowerShell; use `--shell powershell` to target explicitly *(Feb 28)*

## Improvements

- [ ] Sort select menu by most recently used (instead of alphabetical)
- [ ] `places rename <old> <new>` command
- [ ] Tab completion for place names (bash/zsh/PowerShell)
- [ ] Validate that saved paths still exist on `list` / `select` (warn if missing)
