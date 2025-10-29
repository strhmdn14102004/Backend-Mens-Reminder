package main

import (
	"log"
	"net/http"

	"backend_mens/internal/config"
	"backend_mens/internal/db"
	"backend_mens/internal/httpserver"
	"backend_mens/internal/scheduler"
	"backend_mens/internal/telegram"
)

func main() {
	cfg := config.Load()
	d, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// scheduler
	s := &scheduler.Service{
		DB:  d,
		Bot: &telegram.Bot{Token: cfg.TelegramToken},
	}
	c := s.Start()
	defer c.Stop()

	h := httpserver.New(d, cfg.JWTSecret, cfg.BaseURL, cfg.TelegramToken, s)

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, h))
}
