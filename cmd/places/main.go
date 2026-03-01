package main

import (
	"fmt"
	"os"
)

// fatal prints an error message to stderr and exits with code 1.
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "add":
		name := ""
		path := ""
		if len(args) >= 2 {
			name = args[1]
		}
		if len(args) >= 3 {
			path = args[2]
		}
		cmdAdd(name, path)
	case "list", "ls":
		hasJSON := false
		for _, a := range args[1:] {
			if a == "--json" {
				hasJSON = true
			}
		}
		if hasJSON {
			cmdListJSON()
		} else {
			cmdList()
		}
	case "where":
		cmdWhere()
	case "exists":
		if len(args) < 2 {
			fatal("expected: places exists <name>")
		}
		cmdExists(args[1])
	case "select":
		cmdSelect()
	case "go":
		if len(args) < 2 {
			fatal("expected: places go <name>")
		}
		cmdGo(args[1])
	case "rm":
		if len(args) < 2 {
			fatal("expected: places rm <name>")
		}
		cmdRm(args[1])
	case "rename", "mv":
		if len(args) < 3 {
			fatal("expected: places rename <old> <new>")
		}
		cmdRename(args[1], args[2])
	case "code":
		if len(args) < 2 {
			fatal("expected: places code <name>")
		}
		cmdCode(args[1])
	case "shell":
		if len(args) < 2 {
			fatal("expected: places shell <name>")
		}
		cmdShell(args[1])
	case "autostart":
		arg := ""
		if len(args) >= 2 {
			arg = args[1]
		}
		cmdAutostart(arg)
	case "stats":
		cmdStats()
	case "prune":
		cmdPrune()
	case "app":
		cmdApp()
	case "edit":
		editor := ""
		if len(args) >= 2 {
			editor = args[1]
		}
		cmdEdit(editor)
	case "init":
		cmdInit()
	case "shell-hook":
		shellHookCmd(args[1:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`places - Directory bookmarks for quick navigation

Usage:
  places add [name] [path]     Save current dir (or given path) with a shortcut name
                               If name is omitted, uses the directory basename
  places list [--json]         List all saved places (alias: ls)
  places select                Browse and pick a place (sorted by recent use)
  places go <name>             Print the path for a place (used by shell wrapper)
  places rm <name>             Remove a saved place
  places rename <old> <new>    Rename a saved place (alias: mv)
  places stats                 Show usage summary
  places code <name>            Open a place in VS Code
  places shell <name>          Open a new terminal at a place (no hook needed)
  places where                 Print the place name for the current directory
  places exists <name>         Exit 0 if a place exists, 1 otherwise
  places autostart [on|off]    Enable/disable starting tray app on login (Windows)
  places prune                 Remove places where the directory no longer exists
  places app                   Open the places desktop app
  places edit [editor]         Open places.json in $EDITOR (or specified editor)
  places init                  Set up shell hooks (auto-detects shell, installs all)
  places shell-hook install    Install p() function (auto-detects shell)
  places shell-hook uninstall  Remove p() function from shell config
  places help                  Show this help message

Options:
  --shell bash|zsh|powershell|cmd  Override auto-detected shell (for shell-hook)

Shell integration:
  places cannot change your shell's directory directly (child process
  limitation). The shell hook installs a 'p' wrapper that calls places
  and performs the actual cd/Set-Location.

  After installing, use:
    p <name>    Jump to a saved place
    p           Browse and select interactively

Setup for Bash/Zsh:
  1. places shell-hook install
  2. source ~/.bashrc   (or ~/.zshrc)

Setup for PowerShell:
  1. Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
     (one-time, allows loading your profile script)
  2. places shell-hook install
  3. . $PROFILE
     (or restart PowerShell)

Setup for cmd.exe:
  1. places shell-hook install --shell cmd
     (creates p.bat next to places.exe)

To install for multiple shells, use --shell:
  places shell-hook install --shell bash
  places shell-hook install --shell powershell
  places shell-hook install --shell cmd

Created by Thomas Häuser
https://mavwarf.netlify.app/
https://github.com/Mavwarf/places`)
}
