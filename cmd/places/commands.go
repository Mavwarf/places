package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Mavwarf/places/internal/config"
	"github.com/Mavwarf/places/internal/launcher"
)

func cmdAdd(name, path string, tags []string) {
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

	place := &config.Place{
		Path:    path,
		AddedAt: time.Now(),
	}
	for _, t := range tags {
		config.AddTag(place, t)
	}
	cfg.Places[name] = place

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	tagStr := ""
	if len(place.Tags) > 0 {
		tagStr = fmt.Sprintf(" %s[%s]%s", colorDim, strings.Join(place.Tags, ", "), colorReset)
	}
	fmt.Printf("Saved %q -> %s%s\n", name, path, tagStr)
}

func cmdList(tagFilter string, favOnly bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fmt.Println("No places saved. Use 'places add <name>' to save one.")
		return
	}

	names := config.SortedNames(cfg)

	// Filter by tag if specified.
	if tagFilter != "" {
		var filtered []string
		for _, name := range names {
			for _, t := range cfg.Places[name].Tags {
				if t == tagFilter {
					filtered = append(filtered, name)
					break
				}
			}
		}
		names = filtered
		if len(names) == 0 {
			fmt.Printf("No places with tag %q.\n", tagFilter)
			return
		}
	}

	// Filter by favorites if requested.
	if favOnly {
		var filtered []string
		for _, name := range names {
			if cfg.Places[name].Favorite {
				filtered = append(filtered, name)
			}
		}
		names = filtered
		if len(names) == 0 {
			fmt.Println("No favorite places. Use 'places fav <name>' to mark one.")
			return
		}
	}

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
		tagBadge := ""
		if len(p.Tags) > 0 {
			tagBadge = fmt.Sprintf(" %s[%s]%s", colorDim, strings.Join(p.Tags, ", "), colorReset)
		}
		star := ""
		if p.Favorite {
			star = fmt.Sprintf("%s★%s ", colorYellow, colorReset)
		}
		deskBadge := ""
		if p.Desktop > 0 {
			deskBadge = fmt.Sprintf(" %s[D%d]%s", colorDim, p.Desktop, colorReset)
		}
		fmt.Printf("  %s%s%-*s%s  %s%s%s  %s%s%s%s\n", star, colorGreen, maxLen, name, colorReset, colorCyan, p.Path, colorReset, stats, tagBadge, deskBadge, warning)
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
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "places: warning: %v\n", err)
	}

	// Print path to stdout for the shell wrapper to capture.
	fmt.Print(place.Path)
}

func cmdCode(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		place, name = fuzzyFind(cfg, name)
		if place == nil {
			fatal("unknown place %q", name)
		}
	}

	if _, err := os.Stat(place.Path); err != nil {
		fatal("directory does not exist: %s", place.Path)
	}

	if err := launcher.Detach(launcher.Code(place.Path)); err != nil {
		fatal("cannot start VS Code: %v", err)
	}
}

func cmdShell(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		place, name = fuzzyFind(cfg, name)
		if place == nil {
			fatal("unknown place %q", name)
		}
	}

	if _, err := os.Stat(place.Path); err != nil {
		fatal("directory does not exist: %s", place.Path)
	}

	cmd := launcher.PowerShell(place.Path)
	if runtime.GOOS == "windows" {
		if err := cmd.Start(); err != nil {
			fatal("cannot start shell: %v", err)
		}
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatal("shell exited with error: %v", err)
		}
	}
}

func cmdAutostart(arg string) {
	if runtime.GOOS != "windows" {
		fatal("autostart is only supported on Windows")
	}

	// Find places-app.exe next to the places binary.
	exe, err := os.Executable()
	if err != nil {
		fatal("cannot determine binary path: %v", err)
	}
	exe, _ = filepath.Abs(exe)
	appExe := filepath.Join(filepath.Dir(exe), "places-app.exe")

	const regKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	const regName = "places"

	switch arg {
	case "on":
		if _, err := os.Stat(appExe); err != nil {
			fatal("places-app not found at %s", appExe)
		}
		cmd := exec.Command("reg", "add", regKey, "/v", regName, "/t", "REG_SZ", "/d", appExe, "/f")
		if out, err := cmd.CombinedOutput(); err != nil {
			fatal("failed to enable autostart: %s", strings.TrimSpace(string(out)))
		}
		fmt.Println("Autostart enabled — places-app will start on login.")
	case "off":
		cmd := exec.Command("reg", "delete", regKey, "/v", regName, "/f")
		if out, err := cmd.CombinedOutput(); err != nil {
			fatal("failed to disable autostart: %s", strings.TrimSpace(string(out)))
		}
		fmt.Println("Autostart disabled.")
	default:
		// Show status.
		cmd := exec.Command("reg", "query", regKey, "/v", regName)
		if err := cmd.Run(); err != nil {
			fmt.Println("Autostart: off")
		} else {
			fmt.Println("Autostart: on")
		}
	}
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

func cmdApp() {
	// Look for places-app next to the places binary.
	exe, err := os.Executable()
	if err != nil {
		fatal("cannot determine binary path: %v", err)
	}
	exe, _ = filepath.Abs(exe)
	appExe := filepath.Join(filepath.Dir(exe), "places-app.exe")
	if runtime.GOOS != "windows" {
		appExe = filepath.Join(filepath.Dir(exe), "places-app")
	}
	if _, err := os.Stat(appExe); err != nil {
		fatal("places-app not found at %s", appExe)
	}
	cmd := exec.Command(appExe)
	if err := cmd.Start(); err != nil {
		fatal("cannot start places-app: %v", err)
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

	// Build items for interactive selector.
	items := make([]selectItem, len(names))
	for i, name := range names {
		warning := ""
		if _, err := os.Stat(cfg.Places[name].Path); err != nil {
			warning = "[missing!]"
		}
		items[i] = selectItem{Name: name, Path: cfg.Places[name].Path, Warning: warning}
	}

	idx, ok, err := runInteractiveSelect(items)
	if err != nil {
		fatal("failed to enable interactive mode: %v", err)
	}
	if !ok {
		os.Exit(1)
	}

	selected := cfg.Places[names[idx]]
	config.RecordUse(selected)
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "places: warning: %v\n", err)
	}

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

func cmdPrune() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	var pruned []string
	for name, p := range cfg.Places {
		if _, err := os.Stat(p.Path); err != nil {
			pruned = append(pruned, name)
			delete(cfg.Places, name)
		}
	}

	if len(pruned) == 0 {
		fmt.Println("Nothing to prune — all directories exist.")
		return
	}

	sort.Strings(pruned)
	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	for _, name := range pruned {
		fmt.Printf("Removed %s%s%s (directory missing)\n", colorYellow, name, colorReset)
	}
	fmt.Printf("Pruned %d place(s).\n", len(pruned))
}

func cmdWhere() {
	cwd, err := os.Getwd()
	if err != nil {
		fatal("cannot determine current directory: %v", err)
	}
	cwd = filepath.Clean(cwd)

	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	for name, p := range cfg.Places {
		if strings.EqualFold(filepath.Clean(p.Path), cwd) {
			fmt.Println(name)
			return
		}
	}

	os.Exit(1)
}

func cmdExists(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if _, ok := cfg.Places[name]; ok {
		os.Exit(0)
	}
	os.Exit(1)
}

func cmdListJSON(tagFilter string, favOnly bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	type jsonPlace struct {
		Name       string   `json:"name"`
		Path       string   `json:"path"`
		UseCount   int      `json:"use_count"`
		AddedAt    string   `json:"added_at"`
		LastUsedAt string   `json:"last_used_at,omitempty"`
		Tags       []string `json:"tags,omitempty"`
		Favorite   bool     `json:"favorite,omitempty"`
		Desktop    int      `json:"desktop,omitempty"`
	}

	names := config.SortedNames(cfg)
	places := make([]jsonPlace, 0, len(names))
	for _, name := range names {
		p := cfg.Places[name]
		// Filter by tag if specified.
		if tagFilter != "" {
			found := false
			for _, t := range p.Tags {
				if t == tagFilter {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Filter by favorites if requested.
		if favOnly && !p.Favorite {
			continue
		}
		jp := jsonPlace{
			Name:     name,
			Path:     p.Path,
			UseCount: p.UseCount,
			AddedAt:  p.AddedAt.Format(time.RFC3339),
			Tags:     p.Tags,
			Favorite: p.Favorite,
			Desktop:  p.Desktop,
		}
		if !p.LastUsedAt.IsZero() {
			jp.LastUsedAt = p.LastUsedAt.Format(time.RFC3339)
		}
		places = append(places, jp)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(places)
}

func cmdDesktop(name string, n int) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	place.Desktop = n

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	if n == 0 {
		fmt.Printf("Cleared desktop for %q\n", name)
	} else {
		fmt.Printf("Set %q to desktop %d\n", name, n)
	}
}

func cmdFav(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	place.Favorite = true

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Marked %q as favorite\n", name)
}

func cmdUnfav(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	place.Favorite = false

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Unmarked %q as favorite\n", name)
}

func cmdTag(name, tag string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	config.AddTag(place, tag)

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Tagged %q with %q\n", name, strings.ToLower(strings.TrimSpace(tag)))
}

func cmdUntag(name, tag string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	if !config.RemoveTag(place, tag) {
		fatal("place %q does not have tag %q", name, tag)
	}

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Removed tag %q from %q\n", strings.ToLower(strings.TrimSpace(tag)), name)
}

func cmdTags() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	tags := config.AllTags(cfg)
	if len(tags) == 0 {
		fmt.Println("No tags. Use 'places tag <name> <tag>' to add one.")
		return
	}

	// Count places per tag.
	counts := make(map[string]int)
	for _, p := range cfg.Places {
		for _, t := range p.Tags {
			counts[t]++
		}
	}

	for _, t := range tags {
		n := counts[t]
		unit := "place"
		if n != 1 {
			unit = "places"
		}
		fmt.Printf("  %s%s%s  %s(%d %s)%s\n", colorGreen, t, colorReset, colorDim, n, unit, colorReset)
	}
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
// Returns nil if no match found. Calls fatal() if multiple ambiguous matches exist.
func fuzzyFind(cfg config.Config, query string) (*config.Place, string) {
	lower := strings.ToLower(query)
	var matches []string
	for name := range cfg.Places {
		if strings.Contains(strings.ToLower(name), lower) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 1 {
		return cfg.Places[matches[0]], matches[0]
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		fatal("ambiguous place %q — matches: %s", query, strings.Join(matches, ", "))
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
