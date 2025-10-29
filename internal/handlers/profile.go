package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"backend_mens/internal/middleware"
	"backend_mens/internal/scheduler"
)

type ProfileHandler struct {
	DB *sql.DB
}

func (h *ProfileHandler) GetSummary(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	uid := r.Context().Value(middleware.UserIDKey).(int64)
	var lastStart time.Time
	var cycleLen int
	err := h.DB.QueryRow(`SELECT last_period_start, cycle_length FROM cycles WHERE user_id=$1`, uid).Scan(&lastStart, &cycleLen)
	if err != nil { http.Error(w, "not found", 404); return }
	s := scheduler.FriendlySummary(lastStart, cycleLen)
	_ = json.NewEncoder(w).Encode(map[string]string{"summary": s})
}
