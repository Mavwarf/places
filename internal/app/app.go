// Package app implements the HTTP server for the desktop dashboard.
// It serves the embedded HTML frontend and provides a JSON API for managing
// places. All state-mutating endpoints use config.Lock/Unlock to serialize
// concurrent read-modify-write cycles (e.g. when the tray and UI both
// modify config simultaneously).

package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Mavwarf/places/internal/config"
	"github.com/Mavwarf/places/internal/desktop"
	"github.com/Mavwarf/places/internal/launcher"
)

// Version is the build version string, injected at compile time via ldflags.
var Version = "dev"

// BuildTime is the build timestamp, injected at compile time via ldflags.
var BuildTime = ""

// defaultActions lists the built-in action names used by handleOpen and handleToggleDefault.
var defaultActions = map[string]bool{
	"powershell": true,
	"cmd":        true,
	"claude":     true,
	"code":       true,
	"explorer":   true,
}

type openURLReq struct {
	URL string `json:"url"`
}

//go:embed static/index.html
var staticFS embed.FS

type jsonPlace struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Exists     bool     `json:"exists"`
	UseCount   int      `json:"use_count"`
	AddedAt    string   `json:"added_at"`
	LastUsedAt string   `json:"last_used_at,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Favorite   bool     `json:"favorite,omitempty"`
	Desktop    int      `json:"desktop,omitempty"`
	Actions        []string `json:"actions,omitempty"`
	Note           string   `json:"note,omitempty"`
	HiddenDefaults []string `json:"hidden_defaults,omitempty"`
}

type actionAssignReq struct {
	Name   string `json:"name"`   // place name
	Action string `json:"action"` // action name
}

type openReq struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Shift  bool   `json:"shift,omitempty"`
	Ctrl   bool   `json:"ctrl,omitempty"`
}

type addReq struct {
	Name string   `json:"name"`
	Path string   `json:"path"`
	Tags []string `json:"tags,omitempty"`
	Note string   `json:"note,omitempty"`
}

type tagReq struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type favReq struct {
	Name     string `json:"name"`
	Favorite bool   `json:"favorite"`
}

type desktopReq struct {
	Name    string `json:"name"`
	Desktop int    `json:"desktop"`
}

type noteReq struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

type renameReq struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

type updatePathReq struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type rmReq struct {
	Name string `json:"name"`
}

// Callbacks bridges HTTP endpoints to Wails window operations.
// Any callback may be nil, in which case its endpoint is not registered.
type Callbacks struct {
	Show     func()                 // bring window to front (single-instance detection)
	Browse   func() (string, error) // open native folder picker dialog
	Minimize func()                 // minimize the window
	Quit     func()                 // fully exit the application (called via goroutine to allow response)
	Topmost  func(bool)             // toggle always-on-top via SetWindowPos
	LastDrop func() string          // retrieve last drag-and-dropped folder path
}

// Serve starts the HTTP server on the given port.
func Serve(port int, cb Callbacks) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/places", handlePlaces)
	mux.HandleFunc("/api/open", handleOpen)
	mux.HandleFunc("/api/rm", handleRm)
	mux.HandleFunc("/api/add", handleAdd)
	mux.HandleFunc("/api/tag", handleTag)
	mux.HandleFunc("/api/untag", handleUntag)
	mux.HandleFunc("/api/fav", handleFav)
	mux.HandleFunc("/api/desktop", handleDesktop)
	mux.HandleFunc("/api/desktop-count", handleDesktopCount)
	mux.HandleFunc("/api/switch-desktop", handleSwitchDesktop)
	mux.HandleFunc("/api/open-url", handleOpenURL)
	mux.HandleFunc("/api/actions", handleActions)
	mux.HandleFunc("/api/run-action", handleRunAction)
	mux.HandleFunc("/api/action-assign", handleActionAssign)
	mux.HandleFunc("/api/action-unassign", handleActionUnassign)
	mux.HandleFunc("/api/note", handleNote)
	mux.HandleFunc("/api/rename", handleRename)
	mux.HandleFunc("/api/update-path", handleUpdatePath)
	mux.HandleFunc("/api/export", handleExport)
	mux.HandleFunc("/api/import", handleImport)
	mux.HandleFunc("/api/git-status", handleGitStatus)
	mux.HandleFunc("/api/setup-notify", handleSetupNotify)
	mux.HandleFunc("/api/notify-path", handleNotifyPath)
	mux.HandleFunc("/api/action-define", handleActionDefine)
	mux.HandleFunc("/api/action-delete", handleActionDelete)
	mux.HandleFunc("/api/default-hidden", handleDefaultHidden)
	mux.HandleFunc("/api/record-recent", handleRecordRecent)
	mux.HandleFunc("/api/sync-recent", handleSyncRecent)
	mux.HandleFunc("/api/toggle-default", handleToggleDefault)
	if cb.Show != nil {
		mux.HandleFunc("/api/show", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			cb.Show()
			w.WriteHeader(http.StatusNoContent)
		})
	}
	if cb.Browse != nil {
		mux.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			path, err := cb.Browse()
			if err != nil {
				http.Error(w, "browse failed", http.StatusInternalServerError)
				return
			}
			if path == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"path": path})
		})
	}
	if cb.Minimize != nil {
		mux.HandleFunc("/api/minimize", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			cb.Minimize()
			w.WriteHeader(http.StatusNoContent)
		})
	}
	if cb.Quit != nil {
		mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			go cb.Quit()
		})
	}
	if cb.Topmost != nil {
		mux.HandleFunc("/api/topmost", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				OnTop bool `json:"on_top"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			cb.Topmost(req.OnTop)
			w.WriteHeader(http.StatusNoContent)
		})
	}

	if cb.LastDrop != nil {
		mux.HandleFunc("/api/last-drop", func(w http.ResponseWriter, r *http.Request) {
			path := cb.LastDrop()
			if path == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"path": path})
		})
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	origin := fmt.Sprintf("http://127.0.0.1:%d", port)
	return http.ListenAndServe(addr, &originGuard{handler: mux, allowed: origin})
}

// originGuard rejects requests with an Origin header that does not match
// the expected local server origin. Browsers always send Origin on cross-origin
// requests, so this blocks malicious websites from hitting the API. Requests
// without an Origin header (e.g. curl, the single-instance POST) are allowed.
type originGuard struct {
	handler http.Handler
	allowed string
}

func (g *originGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && origin != g.allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	g.handler.ServeHTTP(w, r)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	html := strings.Replace(string(data), "{{version}}", Version, 1)
	bt := ""
	if BuildTime != "" {
		bt = BuildTime + " &middot; "
	}
	html = strings.Replace(html, "{{build_time}}", bt, 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// handlePlaces returns the full list of places and action definitions as JSON.
// Response format: {"places": [...], "actions": {"name": {"label": "...", "cmd": "..."}}}
// No config.Lock needed — this is a read-only endpoint with no modify-save cycle.
func handlePlaces(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	names := config.SortedNames(cfg)
	places := make([]jsonPlace, 0, len(names))
	for _, name := range names {
		p := cfg.Places[name]
		_, statErr := os.Stat(p.Path)
		jp := jsonPlace{
			Name:           name,
			Path:           p.Path,
			Exists:         statErr == nil,
			UseCount:       p.UseCount,
			AddedAt:        p.AddedAt.Format(time.RFC3339),
			Tags:           p.Tags,
			Favorite:       p.Favorite,
			Desktop:        p.Desktop,
			Actions:        p.Actions,
			Note:           p.Note,
			HiddenDefaults: p.HiddenDefaults,
		}
		if !p.LastUsedAt.IsZero() {
			jp.LastUsedAt = p.LastUsedAt.Format(time.RFC3339)
		}
		places = append(places, jp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"places":         places,
		"actions":        cfg.Actions,
		"notify_path":    cfg.NotifyPath,
		"default_hidden": cfg.DefaultHidden,
	})
}

// handleOpen launches an application (PowerShell, cmd, Claude, Code, Explorer)
// at a place's directory. Uses manual Lock/Unlock (not defer) because we need
// to release the lock before launching the external process — launches can
// take time and we don't want to block other API calls.
func handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req openReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Lock for the read-modify(RecordUse)-save cycle.
	config.Lock()
	cfg, err := config.Load()
	if err != nil {
		config.Unlock()
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		config.Unlock()
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(place.Path); err != nil {
		config.Unlock()
		http.Error(w, "directory does not exist", http.StatusBadRequest)
		return
	}

	if !defaultActions[req.Action] {
		config.Unlock()
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	config.RecordUse(place)
	if err := config.Save(cfg); err != nil {
		config.Unlock()
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	path := place.Path
	name := req.Name
	desk := place.Desktop
	config.Unlock()

	launcher.SwitchDesktop(desk)

	var cmd *exec.Cmd
	switch req.Action {
	case "powershell":
		cmd = launcher.PowerShell(path)
	case "cmd":
		cmd = launcher.Cmd(path)
	case "claude":
		cmd = launcher.Claude(path, name, req.Shift, req.Ctrl)
	case "code":
		cmd = launcher.Code(path)
	case "explorer":
		cmd = launcher.Explorer(path)
	}

	if err := launcher.Detach(cmd); err != nil {
		http.Error(w, "failed to launch application", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDesktop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req desktopReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	maxDesktop := 4
	if n, err := desktop.Count(); err == nil && n > 0 {
		maxDesktop = n
	}
	if req.Desktop < 0 || req.Desktop > maxDesktop {
		http.Error(w, fmt.Sprintf("desktop must be 0-%d", maxDesktop), http.StatusBadRequest)
		return
	}

	place.Desktop = req.Desktop

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDesktopCount(w http.ResponseWriter, r *http.Request) {
	count := 4
	if n, err := desktop.Count(); err == nil && n > 0 {
		count = n
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

func handleSwitchDesktop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req desktopReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Desktop <= 0 {
		http.Error(w, "no desktop assigned", http.StatusBadRequest)
		return
	}

	launcher.SwitchDesktop(req.Desktop)
	w.WriteHeader(http.StatusNoContent)
}

func handleRm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req rmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	if _, ok := cfg.Places[req.Name]; !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	delete(cfg.Places, req.Name)
	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req addReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Path == "" {
		http.Error(w, "name and path are required", http.StatusBadRequest)
		return
	}

	if err := config.ValidateName(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	req.Path = absPath

	info, err := os.Stat(req.Path)
	if err != nil {
		http.Error(w, "path does not exist", http.StatusBadRequest)
		return
	}
	if !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	if _, ok := cfg.Places[req.Name]; ok {
		http.Error(w, "place already exists", http.StatusConflict)
		return
	}

	place := &config.Place{
		Path:    req.Path,
		AddedAt: time.Now(),
		Note:    req.Note,
	}
	if len(cfg.DefaultHidden) > 0 {
		place.HiddenDefaults = append([]string{}, cfg.DefaultHidden...)
	}
	for _, t := range req.Tags {
		config.AddTag(place, t)
	}
	cfg.Places[req.Name] = place
	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleFav(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req favReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	place.Favorite = req.Favorite

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tagReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	config.AddTag(place, req.Tag)

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleOpenURL opens a URL in the default browser. Restricted to https:// only.
func handleOpenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req openURLReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(req.URL, "https://") {
		http.Error(w, "only https URLs allowed", http.StatusBadRequest)
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", req.URL)
	case "darwin":
		cmd = exec.Command("open", req.URL)
	default:
		cmd = exec.Command("xdg-open", req.URL)
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, "failed to open URL", http.StatusInternalServerError)
		return
	}
	go cmd.Wait()
	w.WriteHeader(http.StatusNoContent)
}

// handleActions returns all defined custom actions as JSON.
func handleActions(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg.Actions)
}

// handleRunAction executes a custom action for a place.
// Uses manual Lock/Unlock to release the lock before launching the process.
func handleRunAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req actionAssignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	cfg, err := config.Load()
	if err != nil {
		config.Unlock()
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		config.Unlock()
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	action, ok := cfg.Actions[req.Action]
	if !ok {
		config.Unlock()
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	config.RecordUse(place)
	if err := config.Save(cfg); err != nil {
		config.Unlock()
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	path := place.Path
	name := req.Name
	desk := place.Desktop
	cmdTpl := action.Cmd
	config.Unlock()

	launcher.SwitchDesktop(desk)

	expanded := launcher.ExpandAction(cmdTpl, path, name)
	cmd := launcher.Shell(expanded)
	if err := launcher.Detach(cmd); err != nil {
		http.Error(w, "failed to launch action", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleActionAssign assigns a custom action to a place.
func handleActionAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req actionAssignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	if _, ok := cfg.Actions[req.Action]; !ok {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	config.AddAction(place, req.Action)

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleActionUnassign removes a custom action from a place.
func handleActionUnassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req actionAssignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	config.RemoveAction(place, req.Action)

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleNote sets or clears a note on a place.
func handleNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req noteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	place.Note = req.Note

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRename changes a place's name (map key).
func handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req renameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.OldName == "" || req.NewName == "" {
		http.Error(w, "old_name and new_name are required", http.StatusBadRequest)
		return
	}

	if err := config.ValidateName(req.NewName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.OldName]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	if _, exists := cfg.Places[req.NewName]; exists {
		http.Error(w, "a place with that name already exists", http.StatusConflict)
		return
	}

	cfg.Places[req.NewName] = place
	delete(cfg.Places, req.OldName)

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdatePath changes a place's directory path.
func handleUpdatePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req updatePathReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Path == "" {
		http.Error(w, "name and path are required", http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	req.Path = absPath

	info, err := os.Stat(req.Path)
	if err != nil {
		http.Error(w, "path does not exist", http.StatusBadRequest)
		return
	}
	if !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	place.Path = req.Path

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleExport returns the full config as a JSON download.
func handleExport(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="places-export.json"`)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(cfg)
}

// handleImport merges places and actions from uploaded JSON (skip existing).
func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB limit
	var incoming config.Config
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	result := mergeConfig(cfg, &incoming)

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// mergeConfig merges incoming places and actions into cfg, skipping existing entries.
func mergeConfig(cfg config.Config, incoming *config.Config) map[string]interface{} {
	added, skipped := 0, 0
	if incoming.Places != nil {
		for name, place := range incoming.Places {
			if place == nil {
				continue
			}
			if err := config.ValidateName(name); err != nil {
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
	}

	actionsAdded, actionsSkipped := 0, 0
	if incoming.Actions != nil {
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
	}

	return map[string]interface{}{
		"places_added":    added,
		"places_skipped":  skipped,
		"actions_added":   actionsAdded,
		"actions_skipped": actionsSkipped,
	}
}

// handleGitStatus returns the current git branch and dirty/clean status for a place.
// Uses manual Lock/Unlock to release before running git commands.
func handleGitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	cfg, err := config.Load()
	if err != nil {
		config.Unlock()
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		config.Unlock()
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}
	path := place.Path
	config.Unlock()

	if _, err := os.Stat(path); err != nil {
		http.Error(w, "directory does not exist", http.StatusBadRequest)
		return
	}

	branch, dirty, err := gitStatus(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"branch": branch,
		"dirty":  dirty,
	})
}

// gitStatus returns the current branch name and dirty flag for a git repo.
func gitStatus(parent context.Context, path string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	branchCmd := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	hideWindow(branchCmd)
	branchOut, err := branchCmd.Output()
	if err != nil {
		return "", false, fmt.Errorf("not a git repository")
	}

	statusCmd := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain")
	hideWindow(statusCmd)
	statusOut, err := statusCmd.Output()
	if err != nil {
		return "", false, fmt.Errorf("failed to read git status")
	}

	branch := strings.TrimSpace(string(branchOut))
	dirty := len(strings.TrimSpace(string(statusOut))) > 0
	return branch, dirty, nil
}

// handleToggleDefault toggles a built-in action's visibility for a place.
func handleToggleDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req actionAssignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !defaultActions[req.Action] {
		http.Error(w, "unknown default action", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	// Toggle: if already hidden, show it; otherwise hide it.
	found := false
	for i, a := range place.HiddenDefaults {
		if a == req.Action {
			place.HiddenDefaults = append(place.HiddenDefaults[:i], place.HiddenDefaults[i+1:]...)
			if len(place.HiddenDefaults) == 0 {
				place.HiddenDefaults = nil
			}
			found = true
			break
		}
	}
	if !found {
		place.HiddenDefaults = append(place.HiddenDefaults, req.Action)
	}

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleUntag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tagReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	if !config.RemoveTag(place, req.Tag) {
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const maxRecent = 8

// handleRecordRecent records a place+action launch in the config's recent list.
func handleRecordRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req config.RecentEntry
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Action == "" {
		http.Error(w, "name and action required", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	// Remove duplicate if exists.
	filtered := make([]config.RecentEntry, 0, len(cfg.Recent))
	for _, e := range cfg.Recent {
		if e.Name == req.Name && e.Action == req.Action && e.Shift == req.Shift && e.Ctrl == req.Ctrl {
			continue
		}
		filtered = append(filtered, e)
	}
	// Prepend new entry.
	cfg.Recent = append([]config.RecentEntry{req}, filtered...)
	if len(cfg.Recent) > maxRecent {
		cfg.Recent = cfg.Recent[:maxRecent]
	}

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSyncRecent replaces the server's recent list with the full list from the frontend.
func handleSyncRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var entries []config.RecentEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(entries) > maxRecent {
		entries = entries[:maxRecent]
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	cfg.Recent = entries
	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleActionDefine creates or overwrites a custom action definition.
func handleActionDefine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name  string `json:"name"`
		Label string `json:"label"`
		Cmd   string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Label == "" || req.Cmd == "" {
		http.Error(w, "name, label, and cmd are required", http.StatusBadRequest)
		return
	}
	if err := config.ValidateName(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	if cfg.Actions == nil {
		cfg.Actions = make(map[string]*config.Action)
	}
	cfg.Actions[req.Name] = &config.Action{Label: req.Label, Cmd: req.Cmd}
	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleActionDelete deletes a custom action and unassigns it from all places.
func handleActionDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	config.Lock()
	defer config.Unlock()

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	if _, ok := cfg.Actions[req.Name]; !ok {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	delete(cfg.Actions, req.Name)
	count := 0
	for _, place := range cfg.Places {
		if config.RemoveAction(place, req.Name) {
			count++
		}
	}

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"unassigned_count": count})
}

// handleDefaultHidden gets or sets the default hidden actions for new places.
func handleDefaultHidden(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := config.Load()
		if err != nil {
			http.Error(w, "failed to load config", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"hidden": cfg.DefaultHidden})

	case http.MethodPost:
		var req struct {
			Hidden []string `json:"hidden"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		config.Lock()
		defer config.Unlock()
		cfg, err := config.Load()
		if err != nil {
			http.Error(w, "failed to load config", http.StatusInternalServerError)
			return
		}
		cfg.DefaultHidden = req.Hidden
		if err := config.Save(cfg); err != nil {
			http.Error(w, "failed to save config", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

var validProfile = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)

// handleSetupNotify creates or merges notify hooks into a place's .claude/settings.json.
func handleSetupNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !validProfile.MatchString(req.Profile) {
		http.Error(w, "invalid profile: use lowercase letters and hyphens only", http.StatusBadRequest)
		return
	}

	config.Lock()
	cfg, err := config.Load()
	if err != nil {
		config.Unlock()
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	place, ok := cfg.Places[req.Name]
	if !ok {
		config.Unlock()
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}
	placePath := place.Path
	notifyPath := cfg.NotifyPath
	config.Unlock()

	if notifyPath == "" {
		http.Error(w, "notify path not configured", http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(notifyPath); err != nil {
		http.Error(w, "notify.exe not found at "+notifyPath, http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(placePath); err != nil {
		http.Error(w, "place directory does not exist", http.StatusBadRequest)
		return
	}

	claudeDir := filepath.Join(placePath, ".claude")
	settingsFile := filepath.Join(claudeDir, "settings.json")

	var settings map[string]interface{}
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			http.Error(w, "failed to parse existing settings.json", http.StatusInternalServerError)
			return
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Check if notify hooks already exist.
	if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
		for _, key := range []string{"Stop", "Notification"} {
			if arr, ok := hooks[key].([]interface{}); ok {
				for _, entry := range arr {
					if m, ok := entry.(map[string]interface{}); ok {
						if inner, ok := m["hooks"].([]interface{}); ok {
							for _, h := range inner {
								if hm, ok := h.(map[string]interface{}); ok {
									if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "notify") {
										http.Error(w, "notify hooks already configured", http.StatusConflict)
										return
									}
								}
							}
						}
					}
				}
			}
		}
	}

	notifyCmd := filepath.ToSlash(notifyPath)
	makeHook := func(action string) []interface{} {
		return []interface{}{
			map[string]interface{}{
				"matcher": "",
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": fmt.Sprintf(`"%s" %s %s`, notifyCmd, req.Profile, action),
						"timeout": 10,
					},
				},
			},
		}
	}

	hooks, ok2 := settings["hooks"].(map[string]interface{})
	if !ok2 {
		hooks = make(map[string]interface{})
	}
	hooks["Stop"] = makeHook("done")
	hooks["Notification"] = makeHook("attention")
	settings["hooks"] = hooks

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		http.Error(w, "failed to create .claude directory", http.StatusInternalServerError)
		return
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		http.Error(w, "failed to marshal settings", http.StatusInternalServerError)
		return
	}
	out = append(out, '\n')
	if err := os.WriteFile(settingsFile, out, 0644); err != nil {
		http.Error(w, "failed to write settings.json", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleNotifyPath gets or sets the notify.exe path in config.
func handleNotifyPath(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := config.Load()
		if err != nil {
			http.Error(w, "failed to load config", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": cfg.NotifyPath})

	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Path != "" {
			if _, err := os.Stat(req.Path); err != nil {
				http.Error(w, "file not found: "+req.Path, http.StatusBadRequest)
				return
			}
		}
		config.Lock()
		defer config.Unlock()
		cfg, err := config.Load()
		if err != nil {
			http.Error(w, "failed to load config", http.StatusInternalServerError)
			return
		}
		cfg.NotifyPath = req.Path
		if err := config.Save(cfg); err != nil {
			http.Error(w, "failed to save config", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
