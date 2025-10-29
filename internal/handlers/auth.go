package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"backend_mens/internal/middleware"
	"backend_mens/internal/otp"

	"github.com/julienschmidt/httprouter"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

// ========== SEND OTP ==========
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	type req struct {
		Phone string `json:"phone"`
	}
	var in req
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Phone) == "" {
		http.Error(w, "bad request", 400)
		return
	}
	code, _ := otp.Generate(6)
	_ = otp.CreateOTP(r.Context(), h.DB, in.Phone, code, 5*time.Minute)
	_ = otp.SendSMS(in.Phone, "Kode OTP kamu: "+code)
	w.WriteHeader(204)
}

// ========== VERIFY OTP ==========
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	type req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	var in req
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	ok, err := otp.VerifyOTP(context.Background(), h.DB, in.Phone, in.Code)
	if err != nil || !ok {
		http.Error(w, "invalid otp", 401)
		return
	}

	var uid int64
	err = h.DB.QueryRow(`INSERT INTO users (phone) VALUES ($1)
		ON CONFLICT (phone) DO UPDATE SET updated_at=NOW()
		RETURNING id`, in.Phone).Scan(&uid)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}

	token, _ := middleware.IssueJWT(h.JWTSecret, uid)
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// ========== COMPLETE PROFILE ==========
func (h *AuthHandler) CompleteProfile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	uid := r.Context().Value(middleware.UserIDKey).(int64)
	type req struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		BirthDate       string `json:"birth_date"`
		LastPeriodStart string `json:"last_period_start"`
		CycleLength     int    `json:"cycle_length"`
		PeriodLength    int    `json:"period_length"`
	}
	var in req
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	_, err := h.DB.Exec(`UPDATE users SET name=$1, email=$2, birth_date=$3, updated_at=NOW() WHERE id=$4`,
		in.Name, in.Email, in.BirthDate, uid)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO cycles (user_id, last_period_start, cycle_length, period_length)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (user_id)
		DO UPDATE SET last_period_start=EXCLUDED.last_period_start, cycle_length=EXCLUDED.cycle_length, period_length=EXCLUDED.period_length, updated_at=NOW()`,
		uid, in.LastPeriodStart, in.CycleLength, in.PeriodLength)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}
	w.WriteHeader(204)
}
