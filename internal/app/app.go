// Package app implements the HTTP server for the desktop dashboard.
// It serves the embedded HTML frontend and provides a JSON API for managing
// places. All state-mutating endpoints use config.Lock/Unlock to serialize
// concurrent read-modify-write cycles (e.g. when the tray and UI both
// modify config simultaneously).

package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mavwarf/places/internal/config"
	"github.com/Mavwarf/places/internal/launcher"
)

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

// Serve starts the HTTP server on the given port.
// Callback functions bridge HTTP endpoints to Wails window operations:
//   - showFn: bring window to front (single-instance detection)
//   - browseFn: open native folder picker dialog
//   - minimizeFn: minimize the window
//   - quitFn: fully exit the application (called via goroutine to allow response)
//   - topmostFn: toggle always-on-top via SetWindowPos
//   - dropFn: retrieve last drag-and-dropped folder path
//
// Any callback may be nil, in which case its endpoint is not registered.
func Serve(port int, showFn func(), browseFn func() (string, error), minimizeFn func(), quitFn func(), topmostFn func(bool), dropFn func() string) error {
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
	mux.HandleFunc("/api/toggle-default", handleToggleDefault)
	if showFn != nil {
		mux.HandleFunc("/api/show", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			showFn()
			w.WriteHeader(http.StatusNoContent)
		})
	}
	if browseFn != nil {
		mux.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			path, err := browseFn()
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
	if minimizeFn != nil {
		mux.HandleFunc("/api/minimize", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			minimizeFn()
			w.WriteHeader(http.StatusNoContent)
		})
	}
	if quitFn != nil {
		mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			go quitFn()
		})
	}
	if topmostFn != nil {
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
			topmostFn(req.OnTop)
			w.WriteHeader(http.StatusNoContent)
		})
	}

	if dropFn != nil {
		mux.HandleFunc("/api/last-drop", func(w http.ResponseWriter, r *http.Request) {
			path := dropFn()
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
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
		"places":  places,
		"actions": cfg.Actions,
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

	switch req.Action {
	case "powershell", "cmd", "claude", "code", "explorer":
		// valid
	default:
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
		cmd = launcher.Claude(path, name, req.Shift)
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

	if req.Desktop < 0 || req.Desktop > 4 {
		http.Error(w, "desktop must be 0-4", http.StatusBadRequest)
		return
	}

	place.Desktop = req.Desktop

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		cmd = exec.Command("cmd", "/c", "start", "", req.URL)
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

	added := 0
	skipped := 0

	if incoming.Places != nil {
		for name, place := range incoming.Places {
			if place == nil {
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

	actionsAdded := 0
	actionsSkipped := 0
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

	if err := config.Save(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"places_added":    added,
		"places_skipped":  skipped,
		"actions_added":   actionsAdded,
		"actions_skipped": actionsSkipped,
	})
}

// handleGitStatus returns the current git branch and dirty/clean status for a place.
// Uses manual Lock/Unlock to release before running git commands.
func handleGitStatus(w http.ResponseWriter, r *http.Request) {
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

	branchCmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	hideWindow(branchCmd)
	branchOut, err := branchCmd.Output()
	if err != nil {
		http.Error(w, "not a git repository", http.StatusBadRequest)
		return
	}
	branch := strings.TrimSpace(string(branchOut))

	statusCmd := exec.Command("git", "-C", path, "status", "--porcelain")
	hideWindow(statusCmd)
	statusOut, err := statusCmd.Output()
	if err != nil {
		http.Error(w, "failed to read git status", http.StatusInternalServerError)
		return
	}
	dirty := len(strings.TrimSpace(string(statusOut))) > 0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"branch": branch,
		"dirty":  dirty,
	})
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

	switch req.Action {
	case "claude", "explorer", "code", "powershell", "cmd":
		// valid
	default:
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
