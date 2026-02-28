package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	markerBegin   = "# BEGIN places shell-hook"
	markerEnd     = "# END places shell-hook"
	psMarkerBegin = "# BEGIN places shell-hook"
	psMarkerEnd   = "# END places shell-hook"
)

const bashSnippet = `# BEGIN places shell-hook
p() {
  if [ $# -eq 0 ]; then
    local dir
    dir=$(command places select)
    if [ $? -eq 0 ] && [ -n "$dir" ]; then
      cd "$dir" || return
    fi
    return
  fi
  case "$1" in
    add|rm|list|ls|select|help|shell-hook)
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
    if ($args.Count -eq 0) {
        $dir = & places select
        if ($LASTEXITCODE -eq 0 -and $dir) {
            Set-Location $dir
        }
        return
    }
    $cmds = @('add','rm','list','ls','select','help','shell-hook')
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
				fatal("--shell requires a value (bash, zsh, powershell)")
			}
		} else {
			rest = append(rest, args[i])
		}
	}

	if len(rest) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: places shell-hook <install|uninstall> [--shell bash|zsh|powershell]\n")
		os.Exit(1)
	}

	sh := resolveShell(shellOverride)

	switch rest[0] {
	case "install":
		shellHookInstall(sh)
	case "uninstall":
		shellHookUninstall(sh)
	default:
		fmt.Fprintf(os.Stderr, "Unknown shell-hook subcommand: %s\n", rest[0])
		fmt.Fprintf(os.Stderr, "Usage: places shell-hook <install|uninstall>\n")
		os.Exit(1)
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

func shellHookInstall(sh string) {
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

func shellHookUninstall(sh string) {
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
