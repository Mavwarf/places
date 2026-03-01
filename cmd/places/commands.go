package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mavwarf/places/internal/config"
)

func cmdAdd(name, path string) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fatal("cannot determine current directory: %v", err)
		}
	} else {
		// Resolve to absolute path.
		abs, err := filepath.Abs(path)
		if err != nil {
			fatal("cannot resolve path: %v", err)
		}
		path = abs
	}

	// Auto-derive name from directory basename if not provided.
	if name == "" {
		name = filepath.Base(path)
		if name == "." || name == "/" || name == "\\" {
			fatal("cannot derive name from path %q, please provide a name", path)
		}
	}

	// Verify the path exists.
	info, err := os.Stat(path)
	if err != nil {
		fatal("path does not exist: %s", path)
	}
	if !info.IsDir() {
		fatal("path is not a directory: %s", path)
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	cfg.Places[name] = &config.Place{
		Path:    path,
		AddedAt: time.Now(),
	}

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Saved %q -> %s\n", name, path)
}

func cmdList() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fmt.Println("No places saved. Use 'places add <name>' to save one.")
		return
	}

	names := sortedNames(cfg)

	// Find max name length for alignment.
	maxLen := 0
	for _, name := range names {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	for _, name := range names {
		p := cfg.Places[name]
		stats := formatStats(p)
		warning := ""
		if _, err := os.Stat(p.Path); err != nil {
			warning = fmt.Sprintf(" %s[missing!]%s", colorYellow, colorReset)
		}
		fmt.Printf("  %s%-*s%s  %s%s%s  %s%s\n", colorGreen, maxLen, name, colorReset, colorCyan, p.Path, colorReset, stats, warning)
	}
}

func cmdGo(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		// Try fuzzy (substring) match.
		place, name = fuzzyFind(cfg, name)
		if place == nil {
			fatal("unknown place %q", name)
		}
	}

	config.RecordUse(place)
	config.Save(cfg)

	// Print path to stdout for the shell wrapper to capture.
	fmt.Print(place.Path)
}

func cmdEdit(editorOverride string) {
	p, err := config.ConfigPath()
	if err != nil {
		fatal("%v", err)
	}

	editor := editorOverride
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}

	cmd := exec.Command(editor, p)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("editor exited with error: %v", err)
	}
}

func cmdSelect() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fatal("no places saved. Use 'places add <name>' to save one.")
	}

	// Sort by most recently used for select.
	names := sortedByRecent(cfg)

	// Find max name length for alignment.
	maxLen := 0
	for _, name := range names {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	// Print menu to stderr (stdout is reserved for the selected path).
	for i, name := range names {
		warning := ""
		if _, err := os.Stat(cfg.Places[name].Path); err != nil {
			warning = " [missing!]"
		}
		fmt.Fprintf(os.Stderr, "  %d) %-*s  %s%s\n", i+1, maxLen, name, cfg.Places[name].Path, warning)
	}
	fmt.Fprintf(os.Stderr, "Select [1-%d]: ", len(names))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		fatal("reading input: %v", err)
	}

	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(names) {
		fatal("invalid selection")
	}

	selected := cfg.Places[names[choice-1]]
	config.RecordUse(selected)
	config.Save(cfg)

	// Print selected path to stdout for shell wrapper to capture.
	fmt.Print(selected.Path)
}

func cmdRm(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if _, ok := cfg.Places[name]; !ok {
		fatal("unknown place %q", name)
	}

	delete(cfg.Places, name)

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Removed %q\n", name)
}

func cmdRename(oldName, newName string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[oldName]
	if !ok {
		fatal("unknown place %q", oldName)
	}

	if _, exists := cfg.Places[newName]; exists {
		fatal("place %q already exists", newName)
	}

	cfg.Places[newName] = place
	delete(cfg.Places, oldName)

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Renamed %q -> %q\n", oldName, newName)
}

func cmdStats() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fmt.Println("No places saved.")
		return
	}

	totalUses := 0
	var mostUsedName, leastUsedName string
	mostUses := -1
	leastUses := -1

	for name, p := range cfg.Places {
		totalUses += p.UseCount
		if mostUses < 0 || p.UseCount > mostUses {
			mostUses = p.UseCount
			mostUsedName = name
		}
		if leastUses < 0 || p.UseCount < leastUses {
			leastUses = p.UseCount
			leastUsedName = name
		}
	}

	fmt.Printf("Places: %d\n", len(cfg.Places))
	fmt.Printf("Total uses: %d\n", totalUses)
	if mostUses > 0 {
		fmt.Printf("Most used: %s (%d uses)\n", mostUsedName, mostUses)
	}
	if leastUsedName != mostUsedName {
		fmt.Printf("Least used: %s (%d uses)\n", leastUsedName, leastUses)
	}
}

// sortedNames returns place names sorted alphabetically.
func sortedNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Places))
	for name := range cfg.Places {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sortedByRecent returns place names sorted by last used (most recent first),
// with never-used places at the end sorted alphabetically.
func sortedByRecent(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Places))
	for name := range cfg.Places {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		pi := cfg.Places[names[i]]
		pj := cfg.Places[names[j]]
		// Both never used: sort alphabetically.
		if pi.UseCount == 0 && pj.UseCount == 0 {
			return names[i] < names[j]
		}
		// Never-used goes last.
		if pi.UseCount == 0 {
			return false
		}
		if pj.UseCount == 0 {
			return true
		}
		// Most recent first.
		return pi.LastUsedAt.After(pj.LastUsedAt)
	})
	return names
}

// fuzzyFind returns the first place whose name contains the query as a substring.
// Returns nil if no match or multiple ambiguous matches.
func fuzzyFind(cfg config.Config, query string) (*config.Place, string) {
	query = strings.ToLower(query)
	var matchName string
	var matchPlace *config.Place
	count := 0
	for name, place := range cfg.Places {
		if strings.Contains(strings.ToLower(name), query) {
			matchName = name
			matchPlace = place
			count++
		}
	}
	if count == 1 {
		return matchPlace, matchName
	}
	return nil, query
}

// ANSI color helpers.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
	colorYellow = "\033[33m"
)

// formatStats returns a short stats string like "(added Feb 28, 5 uses, last: Feb 28)".
func formatStats(p *config.Place) string {
	added := p.AddedAt.Format("Jan _2")
	if p.UseCount == 0 {
		return fmt.Sprintf("%s(added %s, never used)%s", colorDim, added, colorReset)
	}
	last := p.LastUsedAt.Format("Jan _2 15:04")
	uses := "use"
	if p.UseCount != 1 {
		uses = "uses"
	}
	return fmt.Sprintf("%s(added %s, %d %s, last: %s)%s", colorDim, added, p.UseCount, uses, last, colorReset)
}
