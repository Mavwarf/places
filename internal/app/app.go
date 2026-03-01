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
)

//go:embed static/index.html
var staticFS embed.FS

type jsonPlace struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	UseCount   int    `json:"use_count"`
	AddedAt    string `json:"added_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

type openReq struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type addReq struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type rmReq struct {
	Name string `json:"name"`
}

// Serve starts the HTTP server on the given port.
func Serve(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/places", handlePlaces)
	mux.HandleFunc("/api/open", handleOpen)
	mux.HandleFunc("/api/rm", handleRm)
	mux.HandleFunc("/api/add", handleAdd)

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

	var cmd *exec.Cmd
	switch req.Action {
	case "powershell":
		cmd = exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'", place.Path))
	case "cmd":
		cmd = exec.Command("cmd", "/c", "start", "", "cmd", "/k",
			fmt.Sprintf("cd /d %s", place.Path))
	case "claude":
		cmd = exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'; claude", place.Path))
	case "explorer":
		cmd = exec.Command("explorer", place.Path)
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Detach — don't wait for the process.
	go cmd.Wait()

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

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := cfg.Places[req.Name]; ok {
		http.Error(w, "place already exists", http.StatusConflict)
		return
	}

	cfg.Places[req.Name] = &config.Place{
		Path:    req.Path,
		AddedAt: time.Now(),
	}
	if err := config.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
