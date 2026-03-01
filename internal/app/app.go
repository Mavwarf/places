package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
	return http.ListenAndServe(addr, mux)
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

	places := make([]jsonPlace, 0, len(cfg.Places))
	for name, p := range cfg.Places {
		_, statErr := os.Stat(p.Path)
		jp := jsonPlace{
			Name:     name,
			Path:     p.Path,
			Exists:   statErr == nil,
			UseCount: p.UseCount,
			AddedAt:  p.AddedAt.Format(time.RFC3339),
			Tags:     p.Tags,
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

	if _, err := os.Stat(place.Path); err != nil {
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
	case "explorer":
		fn = launcher.Explorer
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	if err := launcher.Detach(fn(place.Path)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	info, err := os.Stat(req.Path)
	if err != nil {
		http.Error(w, "path does not exist", http.StatusBadRequest)
		return
	}
	if !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}

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
