package handlers

import (
	"backend_mens/internal/middleware"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

type TelegramHandler struct {
	DB        *sql.DB
	BotToken  string
	BaseURL   string
	BotSender interface{ SendMessage(int64, string) error }
}

// === CREATE LINK TOKEN ===
func (h *TelegramHandler) CreateLinkToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	uid := r.Context().Value(middleware.UserIDKey).(int64)

	token := fmt.Sprintf("L%v_%d", time.Now().Unix(), uid)

	_, err := h.DB.Exec(`INSERT INTO telegram_links (user_id, link_token) VALUES ($1,$2)`, uid, token)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}

	resp := map[string]string{
		"deeplink": fmt.Sprintf("https://t.me/%s?start=%s", "Mens_app_bot", token),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// === WEBHOOK ===
func (h *TelegramHandler) Webhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var update struct {
		Message *struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		return
	}
	if update.Message == nil {
		return
	}

	text := strings.TrimSpace(update.Message.Text)

	if strings.HasPrefix(text, "/start ") {
		token := strings.TrimSpace(strings.TrimPrefix(text, "/start "))
		var uid int64
		var used *time.Time
		err := h.DB.QueryRow(`SELECT user_id, consumed_at FROM telegram_links WHERE link_token=$1`, token).Scan(&uid, &used)
		if err != nil || used != nil {
			h.BotSender.SendMessage(update.Message.Chat.ID, "Token tidak valid / terpakai.")
			return
		}
		_, _ = h.DB.Exec(`UPDATE users SET telegram_chat_id=$1 WHERE id=$2`, update.Message.Chat.ID, uid)
		_, _ = h.DB.Exec(`UPDATE telegram_links SET consumed_at=NOW() WHERE link_token=$1`, token)
		h.BotSender.SendMessage(update.Message.Chat.ID, "Tersambung ✅ Reminder aktif.")
	} else {
		h.BotSender.SendMessage(update.Message.Chat.ID, "Gunakan '/start <token>' dari aplikasi.")
	}
}
