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

	if err := config.ValidateName(name); err != nil {
		fatal("%v", err)
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

	if existing, ok := cfg.Places[name]; ok {
		fmt.Fprintf(os.Stderr, "Warning: overwriting %q (was %s)\n", name, existing.Path)
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
	fmt.Fprintf(os.Stderr, "Saved %q -> %s%s\n", name, path, tagStr)
}

func cmdList(tagFilter string, favOnly bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fmt.Fprintln(os.Stderr, "No places saved. Use 'places add <name>' to save one.")
		return
	}

	names := config.FilterNames(cfg, config.SortedNames(cfg), tagFilter, favOnly)
	if len(names) == 0 {
		if tagFilter != "" {
			fmt.Fprintf(os.Stderr, "No places with tag %q.\n", tagFilter)
		} else if favOnly {
			fmt.Fprintln(os.Stderr, "No favorite places. Use 'places fav <name>' to mark one.")
		}
		return
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
		star := "  "
		if p.Favorite {
			star = fmt.Sprintf("%s★%s ", colorYellow, colorReset)
		}
		deskBadge := ""
		if p.Desktop > 0 {
			deskBadge = fmt.Sprintf(" %s[D%d]%s", colorDim, p.Desktop, colorReset)
		}
		fmt.Fprintf(os.Stderr, "  %s%s%-*s%s  %s%s%s  %s%s%s%s\n", star, colorGreen, maxLen, name, colorReset, colorCyan, p.Path, colorReset, stats, tagBadge, deskBadge, warning)
	}
}

// lookupPlace loads config, finds the named place (with fuzzy fallback), and returns both.
func lookupPlace(name string) (config.Config, *config.Place, string) {
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
	return cfg, place, name
}

func cmdGo(name string) {
	cfg, place, _ := lookupPlace(name)

	config.RecordUse(place)
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "places: warning: %v\n", err)
	}

	// Print path to stdout for the shell wrapper to capture.
	fmt.Print(place.Path)
}

func cmdCode(name string) {
	_, place, _ := lookupPlace(name)

	if _, err := os.Stat(place.Path); err != nil {
		fatal("directory does not exist: %s", place.Path)
	}

	if err := launcher.Detach(launcher.Code(place.Path)); err != nil {
		fatal("cannot start VS Code: %v", err)
	}
}

func cmdShell(name string) {
	_, place, _ := lookupPlace(name)

	if _, err := os.Stat(place.Path); err != nil {
		fatal("directory does not exist: %s", place.Path)
	}

	cmd := launcher.PowerShell(place.Path)
	if runtime.GOOS == "windows" {
		// On Windows, PowerShell() uses "cmd /c start" to open a new window,
		// so we just fire-and-forget (Start without Wait).
		if err := cmd.Start(); err != nil {
			fatal("cannot start shell: %v", err)
		}
	} else {
		// On Unix, we replace the current terminal with the new shell session.
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatal("shell exited with error: %v", err)
		}
	}
}

// cmdAutostart manages Windows startup registration via the registry.
// The Run key at HKCU causes programs to launch on user login.
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
		fmt.Fprintln(os.Stderr, "Autostart enabled — places-app will start on login.")
	case "off":
		cmd := exec.Command("reg", "delete", regKey, "/v", regName, "/f")
		if out, err := cmd.CombinedOutput(); err != nil {
			fatal("failed to disable autostart: %s", strings.TrimSpace(string(out)))
		}
		fmt.Fprintln(os.Stderr, "Autostart disabled.")
	default:
		// Show status.
		cmd := exec.Command("reg", "query", regKey, "/v", regName)
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Autostart: off")
		} else {
			fmt.Fprintln(os.Stderr, "Autostart: on")
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
	if err := launcher.StartDetached(cmd); err != nil {
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

	fmt.Fprintf(os.Stderr, "Removed %q\n", name)
}

func cmdRename(oldName, newName string) {
	if err := config.ValidateName(newName); err != nil {
		fatal("%v", err)
	}

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

	fmt.Fprintf(os.Stderr, "Renamed %q -> %q\n", oldName, newName)
}

func cmdStats() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if len(cfg.Places) == 0 {
		fmt.Fprintln(os.Stderr, "No places saved.")
		return
	}

	totalUses := 0
	var mostUsedName, leastUsedName string
	mostUses := -1
	leastUses := -1

	// Iterate in sorted order for deterministic output when counts are equal.
	for _, name := range config.SortedNames(cfg) {
		p := cfg.Places[name]
		totalUses += p.UseCount
		if p.UseCount > mostUses {
			mostUses = p.UseCount
			mostUsedName = name
		}
		if leastUses < 0 || p.UseCount < leastUses {
			leastUses = p.UseCount
			leastUsedName = name
		}
	}

	fmt.Fprintf(os.Stderr, "Places: %d\n", len(cfg.Places))
	fmt.Fprintf(os.Stderr, "Total uses: %d\n", totalUses)
	if mostUses > 0 {
		fmt.Fprintf(os.Stderr, "Most used: %s (%d uses)\n", mostUsedName, mostUses)
	}
	if leastUsedName != mostUsedName {
		fmt.Fprintf(os.Stderr, "Least used: %s (%d uses)\n", leastUsedName, leastUses)
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
		fmt.Fprintln(os.Stderr, "Nothing to prune — all directories exist.")
		return
	}

	sort.Strings(pruned)
	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	for _, name := range pruned {
		fmt.Fprintf(os.Stderr, "Removed %s%s%s (directory missing)\n", colorYellow, name, colorReset)
	}
	fmt.Fprintf(os.Stderr, "Pruned %d place(s).\n", len(pruned))
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

	for _, name := range config.SortedNames(cfg) {
		if strings.EqualFold(filepath.Clean(cfg.Places[name].Path), cwd) {
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
		Actions    []string `json:"actions,omitempty"`
		Note       string   `json:"note,omitempty"`
	}

	names := config.FilterNames(cfg, config.SortedNames(cfg), tagFilter, favOnly)
	places := make([]jsonPlace, 0, len(names))
	for _, name := range names {
		p := cfg.Places[name]
		jp := jsonPlace{
			Name:     name,
			Path:     p.Path,
			UseCount: p.UseCount,
			AddedAt:  p.AddedAt.Format(time.RFC3339),
			Tags:     p.Tags,
			Favorite: p.Favorite,
			Desktop:  p.Desktop,
			Actions:  p.Actions,
			Note:     p.Note,
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
	modifyPlace(name, func(p *config.Place) {
		p.Desktop = n
	})
	if n == 0 {
		fmt.Fprintf(os.Stderr, "Cleared desktop for %q\n", name)
	} else {
		fmt.Fprintf(os.Stderr, "Set %q to desktop %d\n", name, n)
	}
}

func cmdFav(name string) {
	modifyPlace(name, func(p *config.Place) {
		p.Favorite = true
	})
	fmt.Fprintf(os.Stderr, "Marked %q as favorite\n", name)
}

func cmdUnfav(name string) {
	modifyPlace(name, func(p *config.Place) {
		p.Favorite = false
	})
	fmt.Fprintf(os.Stderr, "Unmarked %q as favorite\n", name)
}

func cmdTag(name, tag string) {
	modifyPlace(name, func(p *config.Place) {
		config.AddTag(p, tag)
	})
	fmt.Fprintf(os.Stderr, "Tagged %q with %q\n", name, strings.ToLower(strings.TrimSpace(tag)))
}

func cmdUntag(name, tag string) {
	modifyPlace(name, func(p *config.Place) {
		if !config.RemoveTag(p, tag) {
			fatal("place %q does not have tag %q", name, tag)
		}
	})
	fmt.Fprintf(os.Stderr, "Removed tag %q from %q\n", strings.ToLower(strings.TrimSpace(tag)), name)
}

func cmdTags() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	tags := config.AllTags(cfg)
	if len(tags) == 0 {
		fmt.Fprintln(os.Stderr, "No tags. Use 'places tag <name> <tag>' to add one.")
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
		fmt.Fprintf(os.Stderr, "  %s%s%s  %s(%d %s)%s\n", colorGreen, t, colorReset, colorDim, n, unit, colorReset)
	}
}

func actionCmd(args []string) {
	if len(args) == 0 {
		fatal("Usage: places action <add|rm|list|assign|unassign>")
	}

	switch args[0] {
	case "add":
		name := ""
		label := ""
		cmd := ""
		rest := args[1:]
		for i := 0; i < len(rest); i++ {
			switch rest[i] {
			case "--label":
				if i+1 < len(rest) {
					label = rest[i+1]
					i++
				} else {
					fatal("--label requires a value")
				}
			case "--cmd":
				if i+1 < len(rest) {
					cmd = rest[i+1]
					i++
				} else {
					fatal("--cmd requires a value")
				}
			default:
				if name == "" {
					name = rest[i]
				}
			}
		}
		cmdActionAdd(name, label, cmd)
	case "rm":
		if len(args) < 2 {
			fatal("expected: places action rm <name>")
		}
		cmdActionRm(args[1])
	case "list", "ls":
		cmdActionList()
	case "assign":
		if len(args) < 3 {
			fatal("expected: places action assign <place> <action>")
		}
		cmdActionAssign(args[1], args[2])
	case "unassign":
		if len(args) < 3 {
			fatal("expected: places action unassign <place> <action>")
		}
		cmdActionUnassign(args[1], args[2])
	default:
		fatal("unknown action subcommand: %s\nUsage: places action <add|rm|list|assign|unassign>", args[0])
	}
}

func cmdActionAdd(name, label, cmd string) {
	if name == "" {
		fatal("expected: places action add <name> --label <label> --cmd <cmd>")
	}
	if err := config.ValidateName(name); err != nil {
		fatal("%v", err)
	}
	if label == "" {
		fatal("--label is required")
	}
	if cmd == "" {
		fatal("--cmd is required")
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if _, ok := cfg.Actions[name]; ok {
		fmt.Fprintf(os.Stderr, "Warning: overwriting action %q\n", name)
	}

	cfg.Actions[name] = &config.Action{Label: label, Cmd: cmd}

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Defined action %q (label=%q)\n", name, label)
}

func cmdActionRm(name string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	if _, ok := cfg.Actions[name]; !ok {
		fatal("unknown action %q", name)
	}

	delete(cfg.Actions, name)

	// Remove from all places' action lists.
	for _, place := range cfg.Places {
		config.RemoveAction(place, name)
	}

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Removed action %q\n", name)
}

func cmdActionList() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	names := config.SortedActionNames(cfg)
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "No actions defined. Use 'places action add <name> --label <label> --cmd <cmd>' to define one.")
		return
	}

	for _, name := range names {
		a := cfg.Actions[name]
		fmt.Fprintf(os.Stderr, "  %s%s%s  label=%s%s%s  cmd=%s%s%s\n",
			colorGreen, name, colorReset,
			colorCyan, a.Label, colorReset,
			colorDim, a.Cmd, colorReset)
	}
}

func cmdActionAssign(placeName, actionName string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[placeName]
	if !ok {
		fatal("unknown place %q", placeName)
	}

	if _, ok := cfg.Actions[actionName]; !ok {
		fatal("unknown action %q", actionName)
	}

	config.AddAction(place, actionName)

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Assigned action %q to place %q\n", actionName, placeName)
}

func cmdActionUnassign(placeName, actionName string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[placeName]
	if !ok {
		fatal("unknown place %q", placeName)
	}

	if !config.RemoveAction(place, actionName) {
		fatal("action %q is not assigned to place %q", actionName, placeName)
	}

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Unassigned action %q from place %q\n", actionName, placeName)
}

func cmdNote(name string, text string, clear bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	if clear {
		place.Note = ""
		if err := config.Save(cfg); err != nil {
			fatal("%v", err)
		}
		fmt.Fprintf(os.Stderr, "Cleared note for %q\n", name)
		return
	}

	if text == "" {
		// Print current note to stdout (machine-readable).
		if place.Note == "" {
			fmt.Fprintf(os.Stderr, "No note for %q\n", name)
		} else {
			fmt.Println(place.Note)
		}
		return
	}

	place.Note = text
	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}
	fmt.Fprintf(os.Stderr, "Set note for %q\n", name)
}

func cmdExport() {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		fatal("encoding export: %v", err)
	}
}

func cmdImport(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		fatal("reading import file: %v", err)
	}

	var incoming config.Config
	if err := json.Unmarshal(data, &incoming); err != nil {
		fatal("parsing import file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	added := 0
	skipped := 0

	// Merge places (skip existing and invalid names).
	for name, place := range incoming.Places {
		if place == nil {
			continue
		}
		if config.ValidateName(name) != nil {
			skipped++
			continue
		}
		if _, exists := cfg.Places[name]; exists {
			skipped++
			continue
		}
		cfg.Places[name] = place
		added++
	}

	// Merge actions (skip existing).
	actionsAdded := 0
	actionsSkipped := 0
	for name, action := range incoming.Actions {
		if action == nil {
			continue
		}
		if _, exists := cfg.Actions[name]; exists {
			actionsSkipped++
			continue
		}
		cfg.Actions[name] = action
		actionsAdded++
	}

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Places:  %d added, %d skipped\n", added, skipped)
	fmt.Fprintf(os.Stderr, "Actions: %d added, %d skipped\n", actionsAdded, actionsSkipped)
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

// modifyPlace loads the config, finds the named place, applies fn, and saves.
func modifyPlace(name string, fn func(*config.Place)) {
	cfg, err := config.Load()
	if err != nil {
		fatal("%v", err)
	}

	place, ok := cfg.Places[name]
	if !ok {
		fatal("unknown place %q", name)
	}

	fn(place)

	if err := config.Save(cfg); err != nil {
		fatal("%v", err)
	}
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
