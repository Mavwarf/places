package main

import (
	"fmt"
	"os"
	"strconv"
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
		var tags []string
		rest := args[1:]
		for i := 0; i < len(rest); i++ {
			if rest[i] == "--tag" {
				if i+1 < len(rest) {
					tags = append(tags, rest[i+1])
					i++
				} else {
					fatal("--tag requires a value")
				}
			} else if name == "" {
				name = rest[i]
			} else if path == "" {
				path = rest[i]
			}
		}
		cmdAdd(name, path, tags)
	case "tag":
		if len(args) < 3 {
			fatal("expected: places tag <name> <tag>")
		}
		cmdTag(args[1], args[2])
	case "untag":
		if len(args) < 3 {
			fatal("expected: places untag <name> <tag>")
		}
		cmdUntag(args[1], args[2])
	case "tags":
		cmdTags()
	case "fav":
		if len(args) < 2 {
			fatal("expected: places fav <name>")
		}
		cmdFav(args[1])
	case "unfav":
		if len(args) < 2 {
			fatal("expected: places unfav <name>")
		}
		cmdUnfav(args[1])
	case "list", "ls":
		hasJSON := false
		tagFilter := ""
		favOnly := false
		for i := 1; i < len(args); i++ {
			if args[i] == "--json" {
				hasJSON = true
			} else if args[i] == "--fav" {
				favOnly = true
			} else if args[i] == "--tag" {
				if i+1 < len(args) {
					tagFilter = args[i+1]
					i++
				} else {
					fatal("--tag requires a value")
				}
			}
		}
		if hasJSON {
			cmdListJSON(tagFilter, favOnly)
		} else {
			cmdList(tagFilter, favOnly)
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
	case "desktop":
		if len(args) < 3 {
			fatal("expected: places desktop <name> <0-4>")
		}
		n, err := strconv.Atoi(args[2])
		if err != nil || n < 0 || n > 4 {
			fatal("desktop must be a number 0-4")
		}
		cmdDesktop(args[1], n)
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
    --tag <tag>                Attach a tag (repeatable: --tag work --tag api)
  places fav <name>             Mark a place as favorite
  places unfav <name>           Unmark a place as favorite
  places list [--json]         List all saved places (alias: ls)
    --tag <tag>                Filter by tag
    --fav                      Show only favorites
  places tag <name> <tag>      Add a tag to an existing place
  places untag <name> <tag>    Remove a tag from a place
  places tags                  List all tags with place counts
  places select                Browse and pick a place (sorted by recent use)
  places go <name>             Print the path for a place (used by shell wrapper)
  places rm <name>             Remove a saved place
  places rename <old> <new>    Rename a saved place (alias: mv)
  places stats                 Show usage summary
  places desktop <name> <0-4>   Set virtual desktop for a place (0 = none)
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
