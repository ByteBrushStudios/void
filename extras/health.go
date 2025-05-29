package extras

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
	"void/state"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	System    SystemInfo        `json:"system"`
	Services  map[string]string `json:"services,omitempty"`
	Commit    string            `json:"commit,omitempty"`
}

type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"goroutines"`
	MemoryMB     uint64 `json:"memory_mb"`
}

var startTime = time.Now()

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := GetVoidInfo()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   info.Version,
		Uptime:    time.Since(startTime).String(),
		System: SystemInfo{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			MemoryMB:     bToMb(m.Alloc),
		},
		Commit: info.Commit,
	}

	if len(state.Registry.Services) > 0 {
		response.Services = make(map[string]string)
		for _, service := range state.Registry.Services {
			response.Services[service.Name] = "configured"
		}
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
