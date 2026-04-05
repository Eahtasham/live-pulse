package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
}

func Health(startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Service: "api",
			Status:  "ok",
			Uptime:  time.Since(startTime).Round(time.Second).String(),
		})
	}
}
