package httpserver

import (
	"database/sql"
	"net/http"

	"backend_mens/internal/handlers"
	"backend_mens/internal/middleware"
	"backend_mens/internal/scheduler"
	"backend_mens/internal/telegram"

	"github.com/julienschmidt/httprouter"
)

func New(db *sql.DB, jwtSecret, baseURL, tgToken string, s *scheduler.Service) http.Handler {
	r := httprouter.New()

	auth := &handlers.AuthHandler{DB: db, JWTSecret: jwtSecret}
	prof := &handlers.ProfileHandler{DB: db}
	tg := &handlers.TelegramHandler{
		DB:        db,
		BotToken:  tgToken,
		BaseURL:   baseURL,
		BotSender: &telegram.Bot{Token: tgToken},
	}
	r.GET("/test/reminder", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		s.RunDaily()
		w.Write([]byte("ok"))
	})
	// Test wellness manual
	r.GET("/test/wellness", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		q := r.URL.Query().Get("period")
		if q != "am" && q != "pm" {
			w.WriteHeader(400)
			w.Write([]byte("use ?period=am or ?period=pm"))
			return
		}
		s.RunWellness(q)
		w.Write([]byte("ok"))
	})

	// Auth (public)
	r.POST("/auth/send-otp", auth.SendOTP)
	r.POST("/auth/verify-otp", auth.VerifyOTP)

	// Telegram webhook (public)
	r.POST("/telegram/webhook", tg.Webhook)

	// Protected
	protected := func(h httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			middleware.Auth(jwtSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h(w, r, p)
			})).ServeHTTP(w, r)
		}
	}

	r.POST("/me/complete-profile", protected(auth.CompleteProfile))
	r.POST("/me/telegram/link", protected(tg.CreateLinkToken))
	r.GET("/me/summary", protected(prof.GetSummary))

	return r
}
