# CLAUDE.md

## Build & deploy

```bash
cd cmd/places && go build -o places.exe . && cp places.exe /c/dev/tools/cli/places.exe
cd cmd/places-app && go build -tags production -ldflags "-X main.version=$(git describe --tags --always) -X 'main.buildTime=$(date +%Y-%m-%d %H:%M)' -H windowsgui" -o places-app.exe . && cp places-app.exe /c/dev/tools/cli/places-app.exe
```

## Documentation rules

When adding or changing user-facing features, update all of these together:
- `cmd/places/main.go` `printUsage()` — help output
- `README.md` — usage section and examples
- Shell hook passthrough lists in `shellhook.go` (bash, PowerShell, cmd.exe) — add new commands so `p <cmd>` works
- Wiki at `C:\dev\repos\private\cli_tools\places.wiki\` — CLI-Reference.md, Desktop-App.md, Use-Cases.md as applicable
- `HISTORY.md` — add a one-line summary to the features list and a detailed section under the current date

## Shell hooks

Three shell hook locations must be updated together when the `p()` snippet changes:
- **Bash:** `~/.bashrc`
- **PowerShell Core (pwsh):** auto-detected via `pwsh -NoProfile -Command $PROFILE`
- **Windows PowerShell:** auto-detected via `powershell -NoProfile -Command $PROFILE`
  - Actual path: `C:\Users\thaeu\OneDrive\Dokumente\WindowsPowerShell\Microsoft.PowerShell_profile.ps1`

When adding new commands, add them to all three passthrough lists:
- `cmdBat` — `if /i "%~1"=="<cmd>" goto :passthrough`
- `bashSnippet` — `add|rm|...|<cmd>|...)`
- `psSnippet` — `$cmds = @('add','rm',...,'<cmd>',...)`

After changing snippets in `shellhook.go`, reinstall all hooks:
```bash
places shell-hook uninstall --shell bash && places shell-hook install --shell bash
places shell-hook uninstall --shell powershell && places shell-hook install --shell powershell
```
The Windows PowerShell profile must also be updated manually (separate from pwsh).

## Conventions

- Go 1.24, stdlib only — no external dependencies
- Manual arg parsing with `switch args[0]`, no framework
- `fatal()` helper for stderr + os.Exit(1)
- stdout is reserved for machine-readable output (`go`, `select` print paths only)
- Interactive UI (menus, prompts) goes to stderr
- Config at `~/.config/places/places.json`, auto-created on first use
- Commit messages: imperative mood, short first line
