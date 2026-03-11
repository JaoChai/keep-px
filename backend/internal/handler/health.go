package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type poolStats struct {
	Total    int32 `json:"total"`
	Idle     int32 `json:"idle"`
	Acquired int32 `json:"acquired"`
	Max      int32 `json:"max"`
}

type healthResponse struct {
	Status string     `json:"status"`
	DB     string     `json:"db"`
	Error  string     `json:"error,omitempty"`
	Pool   *poolStats `json:"pool"`
}

type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stat := h.pool.Stat()
	pool := &poolStats{
		Total:    stat.TotalConns(),
		Idle:     stat.IdleConns(),
		Acquired: stat.AcquiredConns(),
		Max:      stat.MaxConns(),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(healthResponse{
			Status: "unhealthy",
			DB:     "unreachable",
			Error:  err.Error(),
			Pool:   pool,
		})
		return
	}

	status := "ok"
	if pool.Max > 0 && pool.Acquired >= int32(float64(pool.Max)*0.8) {
		status = "degraded"
	}

	json.NewEncoder(w).Encode(healthResponse{
		Status: status,
		DB:     "ok",
		Pool:   pool,
	})
}
