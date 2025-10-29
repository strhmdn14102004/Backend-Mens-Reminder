package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"backend_mens/internal/telegram"

	"github.com/robfig/cron/v3"
)

type Service struct {
	DB  *sql.DB
	Bot *telegram.Bot
}

func (s *Service) Start() *cron.Cron {
	c := cron.New() // default: seconds disabled; minutes granularity cukup
	// Jalan tiap hari jam 08:00 WIB (Railway UTC, offset manual bila perlu).
	_, _ = c.AddFunc("0 0 1 * *", func() { // 01:00 UTC ~= 08:00 WIB
		s.RunDaily()
	})
	c.Start()
	return c
}

func (s *Service) RunDaily() {
	ctx := context.Background()
	rows, err := s.DB.QueryContext(ctx, `
		SELECT u.id, u.telegram_chat_id, c.last_period_start, c.cycle_length
		FROM users u
		JOIN cycles c ON c.user_id = u.id
		WHERE u.telegram_chat_id IS NOT NULL`)
	if err != nil {
		return
	}

	defer rows.Close()
	now := time.Now().Truncate(24 * time.Hour)

	for rows.Next() {
		var uid int64
		var chatID *int64
		var lastStart time.Time
		var cycleLen int
		if err := rows.Scan(&uid, &chatID, &lastStart, &cycleLen); err != nil {
			continue
		}
		if chatID == nil {
			continue
		}

		ovulation := lastStart.AddDate(0, 0, cycleLen-14)
		fertileStart := ovulation.AddDate(0, 0, -4)

		nextPeriod := lastStart.AddDate(0, 0, cycleLen)

		switch {
		case sameDay(now, fertileStart), sameDay(now, fertileStart.AddDate(0, 0, 2)):
			_ = s.Bot.SendMessage(*chatID, "🌼 <b>Masa subur</b> akan segera dimulai. Jaga kondisi terbaikmu ya!")
		case sameDay(now, ovulation):
			_ = s.Bot.SendMessage(*chatID, "⭐ <b>Puncak ovulasi hari ini</b> — ini puncak masa subur.")
		case sameDay(now, nextPeriod.AddDate(0, 0, -2)):
			_ = s.Bot.SendMessage(*chatID, "🩸 <b>Perkiraan mens 2 hari lagi</b>. Siapkan keperluanmu ya.")
		}

		// Optional: update siklus bila nextPeriod lewat (otomatis geser window)
		if now.After(nextPeriod) {
			_, _ = s.DB.ExecContext(ctx, `UPDATE cycles SET last_period_start=$1, updated_at=NOW() WHERE user_id=$2`,
				nextPeriod, uid)
		}
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func FriendlySummary(lastStart time.Time, cycleLength int) string {
	ovulation := lastStart.AddDate(0, 0, cycleLength-14)
	fertileStart := ovulation.AddDate(0, 0, -4)
	fertileEnd := ovulation.AddDate(0, 0, 1)
	nextPeriod := lastStart.AddDate(0, 0, cycleLength)
	return fmt.Sprintf("Masa subur: %s–%s | Ovulasi: %s | Prediksi mens berikut: %s",
		fertileStart.Format("02 Jan"), fertileEnd.Format("02 Jan"), ovulation.Format("02 Jan"), nextPeriod.Format("02 Jan"))
}
