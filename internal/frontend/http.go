package frontend

import (
	"encoding/json"
	"net/http"
)

func searchHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Records []string `json:"records,omitempty"`
	}{})
}

func Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /search", searchHandler)

	server := http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}

	return server.ListenAndServe()
}
