package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/utils/log"
)

// Start launches the trigger API server when a token is configured.
// runTask performs one full check; it returns false when a check is
// already running. The token is read from config on every request, so
// config reloads take effect without a restart (the port is fixed at startup).
func Start(runTask func() bool) {
	if config.GlobalConfig.Api.Token == "" {
		return
	}

	port := config.GlobalConfig.Api.Port
	if port <= 0 {
		port = 8799
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		token := config.GlobalConfig.Api.Token
		reqToken := r.URL.Query().Get("token")
		if reqToken == "" {
			reqToken = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if token == "" || subtle.ConstantTimeCompare([]byte(reqToken), []byte(token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		if runTask() {
			log.Info("api trigger accepted, starting check")
			writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
		} else {
			log.Info("api trigger rejected, check already running")
			writeJSON(w, http.StatusConflict, map[string]string{"status": "already_running"})
		}
	})

	server := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info("api server listening on %s", server.Addr)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("api server error: %v", err)
		}
	}()
}

func writeJSON(w http.ResponseWriter, code int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("write api response failed: %v", err)
	}
}
