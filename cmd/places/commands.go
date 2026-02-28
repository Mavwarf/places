package main

import (
	"bufio"
	"fmt"
	"os"
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
		fmt.Printf("  %-*s  %s  %s\n", maxLen, name, p.Path, stats)
	}
}

func cmdGo(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	config.RecordUse(place)
	config.Save(cfg)

	// Print path to stdout for the shell wrapper to capture.
	fmt.Print(place.Path)
}

func cmdSelect() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fatal("no places saved. Use 'places add <name>' to save one.")
	}

	names := sortedNames(cfg)

	// Find max name length for alignment.
	maxLen := 0
	for _, name := range names {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	// Print menu to stderr (stdout is reserved for the selected path).
	for i, name := range names {
		fmt.Fprintf(os.Stderr, "  %d) %-*s  %s\n", i+1, maxLen, name, cfg.Places[name].Path)
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

// sortedNames returns place names sorted alphabetically.
func sortedNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Places))
	for name := range cfg.Places {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// formatStats returns a short stats string like "(added Feb 28, 5 uses, last: Feb 28)".
func formatStats(p *config.Place) string {
	added := p.AddedAt.Format("Jan _2")
	if p.UseCount == 0 {
		return fmt.Sprintf("(added %s, never used)", added)
	}
	last := p.LastUsedAt.Format("Jan _2 15:04")
	uses := "use"
	if p.UseCount != 1 {
		uses = "uses"
	}
	return fmt.Sprintf("(added %s, %d %s, last: %s)", added, p.UseCount, uses, last)
}
