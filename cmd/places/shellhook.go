// shellhook.go — Install/uninstall the `p()` shell wrapper function.
//
// Shell integration strategy:
//   - Bash/Zsh: append a p() function to ~/.bashrc or ~/.zshrc between
//     marker comments (# BEGIN/END places shell-hook) for clean uninstall.
//   - PowerShell (pwsh + Windows PS): same marker approach in $PROFILE.
//   - cmd.exe: write a standalone p.bat next to places.exe (no markers needed;
//     the whole file is owned by us, so uninstall just deletes it).
//
// The p() wrapper handles the "child can't cd parent" problem: `places go`
// prints a path to stdout, and p() captures it and does the actual cd.
// Known commands are passed through directly to avoid the cd logic.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Marker comments that delimit the shell hook block in rc files.
// Uninstall finds and removes everything between these markers.
const (
	markerBegin = "# BEGIN places shell-hook"
	markerEnd   = "# END places shell-hook"
)

// cmdBat is the p.bat wrapper for cmd.exe. It uses a temp file for `select`
// because cmd.exe can't capture stdout into a variable from an interactive
// command that also reads stdin (the `for /f` + stdin conflict).
const cmdBat = `@echo off
if "%~1"=="" goto :select
if /i "%~1"=="select" goto :select
if /i "%~1"=="add" goto :passthrough
if /i "%~1"=="rm" goto :passthrough
if /i "%~1"=="rename" goto :passthrough
if /i "%~1"=="mv" goto :passthrough
if /i "%~1"=="list" goto :passthrough
if /i "%~1"=="ls" goto :passthrough
if /i "%~1"=="code" goto :passthrough
if /i "%~1"=="shell" goto :passthrough
if /i "%~1"=="autostart" goto :passthrough
if /i "%~1"=="stats" goto :passthrough
if /i "%~1"=="where" goto :passthrough
if /i "%~1"=="exists" goto :passthrough
if /i "%~1"=="prune" goto :passthrough
if /i "%~1"=="app" goto :passthrough
if /i "%~1"=="edit" goto :passthrough
if /i "%~1"=="init" goto :passthrough
if /i "%~1"=="help" goto :passthrough
if /i "%~1"=="shell-hook" goto :passthrough
if /i "%~1"=="tag" goto :passthrough
if /i "%~1"=="untag" goto :passthrough
if /i "%~1"=="tags" goto :passthrough
if /i "%~1"=="fav" goto :passthrough
if /i "%~1"=="unfav" goto :passthrough
if /i "%~1"=="desktop" goto :passthrough
if /i "%~1"=="action" goto :passthrough
if /i "%~1"=="note" goto :passthrough
if /i "%~1"=="export" goto :passthrough
if /i "%~1"=="import" goto :passthrough

for /f "delims=" %%d in ('places go "%~1" 2^>nul') do (
    cd /d "%%d"
    goto :eof
)
places go "%~1"
goto :eof

:select
set "_places_tmp=%TEMP%\places_%RANDOM%.tmp"
places select > "%_places_tmp%"
if errorlevel 1 (
    del "%_places_tmp%" 2>nul
    goto :eof
)
set /p "_places_dir=" < "%_places_tmp%"
del "%_places_tmp%" 2>nul
if defined _places_dir cd /d "%_places_dir%"
set "_places_dir="
goto :eof

:passthrough
places %*
`

const bashSnippet = `# BEGIN places shell-hook
p() {
  if [ $# -eq 0 ] || [ "$1" = "select" ]; then
    local dir
    dir=$(command places select)
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir" || return
    fi
    return
  fi
  case "$1" in
    add|rm|rename|mv|list|ls|code|shell|autostart|stats|where|exists|prune|app|edit|init|help|shell-hook|tag|untag|tags|fav|unfav|desktop|action|note|export|import)
      command places "$@"
      return
      ;;
  esac
  local dir
  dir=$(command places go "$1" 2>/dev/null)
  if [ $? -eq 0 ] && [ -n "$dir" ]; then
    cd "$dir" || return
  else
    command places go "$1"
  fi
}
# END places shell-hook`

const psSnippet = `# BEGIN places shell-hook
function p {
    if ($args.Count -eq 0 -or $args[0] -eq 'select') {
        $dir = & places select
        if ($LASTEXITCODE -eq 0 -and $dir) {
            Set-Location $dir
        }
        return
    }
    $cmds = @('add','rm','rename','mv','list','ls','code','shell','autostart','stats','where','exists','prune','app','edit','init','help','shell-hook','tag','untag','tags','fav','unfav','desktop','action','note','export','import')
    if ($cmds -contains $args[0]) {
        & places @args
        return
    }
    $dir = & places go $args[0] 2>$null
    if ($LASTEXITCODE -eq 0 -and $dir) {
        Set-Location $dir
    } else {
        & places go $args[0]
    }
}
# END places shell-hook`

func cmdInit() {
	sh := resolveShell("")
	installed := 0

	// Install shell hook for detected shell.
	if sh == "cmd" {
		if tryInstallCmd() {
			installed++
		}
	} else {
		if tryInstallShell(sh) {
			installed++
		}
		// On Windows, also install cmd hook.
		if runtime.GOOS == "windows" {
			if tryInstallCmd() {
				installed++
			}
		}
	}

	if installed == 0 {
		fmt.Println("places: everything already set up!")
	} else {
		fmt.Println()
		if sh == "powershell" {
			fmt.Println("Next steps:")
			fmt.Println("  1. Ensure execution policy allows profile loading:")
			fmt.Println("     Set-ExecutionPolicy -Scope CurrentUser RemoteSigned")
			fmt.Println("  2. Restart your shell or run: . $PROFILE")
		} else if sh == "cmd" {
			fmt.Println("Next steps:")
			fmt.Println("  Restart cmd.exe to use 'p <name>'")
		} else {
			fmt.Printf("Next steps:\n  Restart your shell or run: source ~/.%src\n", sh)
		}
	}
}

// tryInstallShell installs the hook for a shell (bash/zsh/powershell), skipping if already present.
// Returns true if newly installed.
func tryInstallShell(sh string) bool {
	rcFile, err := resolveRCFile(sh)
	if err != nil {
		fmt.Fprintf(os.Stderr, "places: skipping %s: %v\n", sh, err)
		return false
	}

	existing, _ := os.ReadFile(rcFile)
	if strings.Contains(string(existing), markerBegin) {
		fmt.Printf("places: %s hook already installed in %s (skipped)\n", sh, rcFile)
		return false
	}

	snippet := snippetForShell(sh)

	if err := os.MkdirAll(filepath.Dir(rcFile), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "places: cannot create directory for %s: %v\n", rcFile, err)
		return false
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "places: cannot write to %s: %v\n", rcFile, err)
		return false
	}
	defer f.Close()

	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString("\n" + snippet + "\n")

	fmt.Printf("places: shell hook installed in %s\n", rcFile)
	return true
}

// tryInstallCmd installs p.bat, skipping if already present. Returns true if newly installed.
func tryInstallCmd() bool {
	batPath, err := cmdBatPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "places: skipping cmd: %v\n", err)
		return false
	}

	if _, err := os.Stat(batPath); err == nil {
		fmt.Printf("places: p.bat already exists at %s (skipped)\n", batPath)
		return false
	}

	if err := os.WriteFile(batPath, []byte(cmdBat), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "places: cannot write %s: %v\n", batPath, err)
		return false
	}

	fmt.Printf("places: p.bat installed at %s\n", batPath)
	return true
}

func shellHookCmd(args []string) {
	// Parse --shell flag.
	shellOverride := ""
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--shell" {
			if i+1 < len(args) {
				shellOverride = args[i+1]
				i++
			} else {
				fatal("--shell requires a value (bash, zsh, powershell, cmd)")
			}
		} else {
			rest = append(rest, args[i])
		}
	}

	if len(rest) == 0 {
		fatal("Usage: places shell-hook <install|uninstall> [--shell bash|zsh|powershell|cmd]")
	}

	sh := resolveShell(shellOverride)

	switch rest[0] {
	case "install":
		shellHookInstall(sh)
	case "uninstall":
		shellHookUninstall(sh)
	default:
		fatal("unknown shell-hook subcommand: %s\nUsage: places shell-hook <install|uninstall>", rest[0])
	}
}

func resolveShell(override string) string {
	if override != "" {
		return override
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	sh := os.Getenv("SHELL")
	if strings.HasSuffix(sh, "/zsh") {
		return "zsh"
	}
	return "bash"
}

func resolveRCFile(sh string) (string, error) {
	switch sh {
	case "powershell":
		return psProfilePath()
	case "zsh":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".zshrc"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".bashrc"), nil
	}
}

// psProfilePath detects the PowerShell profile path by asking the shell itself.
// Tries pwsh (PowerShell Core) first, then falls back to powershell (Windows PS).
// -NoProfile prevents loading the existing profile during detection.
func psProfilePath() (string, error) {
	for _, exe := range []string{"pwsh", "powershell"} {
		out, err := exec.Command(exe, "-NoProfile", "-Command", "$PROFILE").Output()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("cannot determine PowerShell $PROFILE path")
}

func snippetForShell(sh string) string {
	switch sh {
	case "powershell":
		return psSnippet
	default:
		return bashSnippet
	}
}

// cmdBatPath returns the path for p.bat next to the places binary.
func cmdBatPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine places binary path: %w", err)
	}
	exe, _ = filepath.Abs(exe)
	return filepath.Join(filepath.Dir(exe), "p.bat"), nil
}

func shellHookInstall(sh string) {
	if sh == "cmd" {
		cmdHookInstall()
		return
	}

	rcFile, err := resolveRCFile(sh)
	if err != nil {
		fatal("%v", err)
	}

	// Check for existing installation.
	existing, _ := os.ReadFile(rcFile)
	if strings.Contains(string(existing), markerBegin) {
		fatal("shell hook already installed in %s (use 'places shell-hook uninstall' first)", rcFile)
	}

	snippet := snippetForShell(sh)

	// Ensure parent directory exists (PowerShell profile dir may not exist).
	if err := os.MkdirAll(filepath.Dir(rcFile), 0755); err != nil {
		fatal("cannot create directory for %s: %v", rcFile, err)
	}

	// Append to rc file (create if needed).
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fatal("cannot write to %s: %v", rcFile, err)
	}
	defer f.Close()

	// Ensure we start on a new line.
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		f.WriteString("\n")
	}

	if _, err := f.WriteString("\n" + snippet + "\n"); err != nil {
		fatal("writing shell hook: %v", err)
	}

	fmt.Printf("places: shell hook installed in %s\n", rcFile)

	switch sh {
	case "powershell":
		fmt.Println("places: restart PowerShell or run: . $PROFILE")
	case "zsh":
		fmt.Println("places: restart your shell or run: source ~/.zshrc")
	default:
		fmt.Println("places: restart your shell or run: source ~/.bashrc")
	}
}

func cmdHookInstall() {
	batPath, err := cmdBatPath()
	if err != nil {
		fatal("%v", err)
	}

	if _, err := os.Stat(batPath); err == nil {
		fatal("p.bat already exists at %s (use 'places shell-hook uninstall --shell cmd' first)", batPath)
	}

	if err := os.WriteFile(batPath, []byte(cmdBat), 0644); err != nil {
		fatal("cannot write %s: %v", batPath, err)
	}

	fmt.Printf("places: p.bat installed at %s\n", batPath)
	fmt.Println("places: use 'p <name>' in cmd.exe to jump to a saved place")
}

func shellHookUninstall(sh string) {
	if sh == "cmd" {
		cmdHookUninstall()
		return
	}

	rcFile, err := resolveRCFile(sh)
	if err != nil {
		fatal("%v", err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			fatal("shell hook not installed (%s does not exist)", rcFile)
		}
		fatal("%v", err)
	}

	content := string(data)
	beginIdx := strings.Index(content, markerBegin)
	if beginIdx < 0 {
		fatal("shell hook not installed in %s (marker not found)", rcFile)
	}

	endIdx := strings.Index(content[beginIdx:], markerEnd)
	if endIdx < 0 {
		fatal("malformed shell hook in %s (begin marker without end marker)", rcFile)
	}
	endIdx += beginIdx + len(markerEnd)

	// Remove the block and any surrounding blank lines.
	before := strings.TrimRight(content[:beginIdx], "\n")
	after := strings.TrimLeft(content[endIdx:], "\n")

	var result string
	if before == "" {
		result = after
	} else if after == "" {
		result = before + "\n"
	} else {
		result = before + "\n\n" + after
	}

	if err := os.WriteFile(rcFile, []byte(result), 0644); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("places: shell hook removed from %s\n", rcFile)
}

func cmdHookUninstall() {
	batPath, err := cmdBatPath()
	if err != nil {
		fatal("%v", err)
	}

	if _, err := os.Stat(batPath); os.IsNotExist(err) {
		fatal("p.bat not found at %s", batPath)
	}

	if err := os.Remove(batPath); err != nil {
		fatal("cannot remove %s: %v", batPath, err)
	}

	fmt.Printf("places: p.bat removed from %s\n", batPath)
}
