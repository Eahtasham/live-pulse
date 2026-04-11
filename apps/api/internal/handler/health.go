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

// Health godoc
// @Summary Health check endpoint
// @Description Returns the health status of the API service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
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
