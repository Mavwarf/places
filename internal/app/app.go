package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Mavwarf/places/internal/config"
	"github.com/Mavwarf/places/internal/launcher"
)

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
}

type openReq struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type addReq struct {
	Name string   `json:"name"`
	Path string   `json:"path"`
	Tags []string `json:"tags,omitempty"`
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

type rmReq struct {
	Name string `json:"name"`
}

// Serve starts the HTTP server on the given port.
// Serve starts the HTTP server on the given port. If showFn is non-nil,
// a POST /api/show endpoint is registered that calls it (used to bring
// the existing window to front when a second instance is launched).
func Serve(port int, showFn func(), browseFn func() (string, error), minimizeFn func(), quitFn func()) error {
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
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

func handlePlaces(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	names := config.SortedNames(cfg)
	places := make([]jsonPlace, 0, len(names))
	for _, name := range names {
		p := cfg.Places[name]
		_, statErr := os.Stat(p.Path)
		jp := jsonPlace{
			Name:     name,
			Path:     p.Path,
			Exists:   statErr == nil,
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(places)
}

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

	config.Lock()
	cfg, err := config.Load()
	if err != nil {
		config.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	var fn func(string) *exec.Cmd
	switch req.Action {
	case "powershell":
		fn = launcher.PowerShell
	case "cmd":
		fn = launcher.Cmd
	case "claude":
		fn = launcher.Claude
	case "code":
		fn = launcher.Code
	case "explorer":
		fn = launcher.Explorer
	default:
		config.Unlock()
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	config.RecordUse(place)
	if err := config.Save(cfg); err != nil {
		config.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	path := place.Path
	desk := place.Desktop
	config.Unlock()

	launcher.SwitchDesktop(desk)

	if err := launcher.Detach(fn(path)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := cfg.Places[req.Name]; !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	delete(cfg.Places, req.Name)
	if err := config.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := cfg.Places[req.Name]; ok {
		http.Error(w, "place already exists", http.StatusConflict)
		return
	}

	place := &config.Place{
		Path:    req.Path,
		AddedAt: time.Now(),
	}
	for _, t := range req.Tags {
		config.AddTag(place, t)
	}
	cfg.Places[req.Name] = place
	if err := config.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	place.Favorite = req.Favorite

	if err := config.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	place, ok := cfg.Places[req.Name]
	if !ok {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}

	config.AddTag(place, req.Tag)

	if err := config.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
