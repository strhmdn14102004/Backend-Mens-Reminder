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

// ---------- Timezone helper (WIB) ----------
func tz() *time.Location {
	// Coba load Asia/Jakarta; fallback ke WIB +07 bila tzdata tidak tersedia di container
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// ---------- Tips harian ----------
var morningTips = []string{
	"🍎 Sarapan tinggi serat (oat + buah) bantu hormon stabil & pencernaan lancar.",
	"🥚 Tambah protein (telur/tahu/tempe) agar energi stabil & gula darah tidak naik-turun.",
	"🫗 Air hangat + sedikit lemon bantu hidrasi & pencernaan.",
	"🥛 Menjelang mens, tambahkan kalsium (susu/almond milk) untuk bantu kurangi kram.",
	"🥄 Biji chia/flax untuk omega-3 yang baik bagi inflamasi ringan.",
}

var eveningTips = []string{
	"🫗 Jangan lupa minum 1–2 gelas air sore ini.",
	"🚶‍♀️ Jalan santai 15–20 menit bantu tidur & sirkulasi.",
	"🥬 Sayur hijau (bayam/kangkung) bantu zat besi terutama saat/menjelang mens.",
	"🧘‍♀️ Peregangan ringan 5 menit bantu kurangi kaku & kram.",
	"🍵 Teh hangat chamomile/jahe untuk relaksasi sebelum tidur.",
}

// Pilih tip deterministik per hari (agar terasa bervariasi tapi tidak lompat-lompat)
func pickTip(pool []string, day int) string {
	if len(pool) == 0 {
		return ""
	}
	return pool[day%len(pool)]
}

// ---------- Start cron ----------
// 07:30 WIB  → Wellness pagi
// 08:00 WIB  → Reminder siklus harian
// 17:30 WIB  → Wellness sore
func (s *Service) Start() *cron.Cron {
	c := cron.New(cron.WithLocation(tz()))
	_, _ = c.AddFunc("CRON_TZ=Asia/Jakarta 30 7 * * *", func() { s.RunWellness("am") })
	_, _ = c.AddFunc("CRON_TZ=Asia/Jakarta 0 8 * * *", func() { s.RunDaily() })
	_, _ = c.AddFunc("CRON_TZ=Asia/Jakarta 30 17 * * *", func() { s.RunWellness("pm") })
	c.Start()
	return c
}

// ---------- Kirim sekali per user/kind/tanggal ----------
// Menggunakan reminder_logs (UNIQUE (user_id, kind, sent_date)) untuk anti-duplikasi.
func (s *Service) sendOnce(chatID int64, uid int64, kind string, text string, now time.Time) {
	sentDate := now.In(tz()).Truncate(24 * time.Hour)
	res, err := s.DB.Exec(
		`INSERT INTO reminder_logs (user_id, kind, sent_date) 
         VALUES ($1,$2,$3) 
         ON CONFLICT DO NOTHING`,
		uid, kind, sentDate,
	)
	if err != nil {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// sudah pernah kirim hari ini untuk kind ini
		return
	}
	_ = s.Bot.SendMessage(chatID, text)
}

// ---------- Wellness ----------
func (s *Service) RunWellness(period string) {
	now := time.Now().In(tz())
	rows, err := s.DB.Query(`
		SELECT u.id, u.telegram_chat_id,
		       COALESCE(p.wellness_am, TRUE), COALESCE(p.wellness_pm, TRUE)
		FROM users u
		LEFT JOIN reminder_prefs p ON p.user_id = u.id
		WHERE u.telegram_chat_id IS NOT NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid int64
		var chatID *int64
		var allowAM, allowPM bool
		if err := rows.Scan(&uid, &chatID, &allowAM, &allowPM); err != nil || chatID == nil {
			continue
		}

		day := now.Day()
		switch period {
		case "am":
			if allowAM {
				tip := pickTip(morningTips, day)
				if tip != "" {
					s.sendOnce(*chatID, uid, "wellness_am", "🌞 Pagi! "+tip, now)
				}
			}
		case "pm":
			if allowPM {
				tip := pickTip(eveningTips, day)
				if tip != "" {
					s.sendOnce(*chatID, uid, "wellness_pm", "🌙 Sore! "+tip, now)
				}
			}
		}
	}
}

// ---------- Reminder Siklus Harian ----------
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

	now := time.Now().In(tz()).Truncate(24 * time.Hour)

	for rows.Next() {
		var uid int64
		var chatID *int64
		var lastStart time.Time
		var cycleLen int
		if err := rows.Scan(&uid, &chatID, &lastStart, &cycleLen); err != nil || chatID == nil {
			continue
		}

		ovulation := lastStart.AddDate(0, 0, cycleLen-14)
		fertileStart := ovulation.AddDate(0, 0, -4)
		fertileEnd := ovulation.AddDate(0, 0, 1) // +1 hari setelah ovulasi
		nextPeriod := lastStart.AddDate(0, 0, cycleLen)

		switch {
		// H-3 & H-1 sebelum mens
		case sameDay(now, nextPeriod.AddDate(0, 0, -3)):
			s.sendOnce(*chatID, uid, "cycle_h-3", "🗓️ <b>Mens H-3</b> — siapkan pembalut & jaga tidur cukup ya.", now)

		case sameDay(now, nextPeriod.AddDate(0, 0, -1)):
			s.sendOnce(*chatID, uid, "cycle_h-1", "🗓️ <b>Mens besok</b> — perbanyak air & pilih makanan ringan.", now)

		// Masa subur & ovulasi
		case sameDay(now, fertileStart), sameDay(now, fertileStart.AddDate(0, 0, 2)):
			s.sendOnce(*chatID, uid, "fertile_start", "🌼 <b>Masa subur</b> mulai. Jaga kondisi terbaikmu ya!", now)

		case sameDay(now, ovulation):
			s.sendOnce(*chatID, uid, "ovulation", "⭐ <b>Puncak ovulasi hari ini</b> — ini puncak masa subur.", now)

		case sameDay(now, fertileEnd):
			s.sendOnce(*chatID, uid, "fertile_end", "🌿 <b>Akhir masa subur</b>. Kembali ke rutinitas & jaga pola makan.", now)

		// Hari-H mens
		case sameDay(now, nextPeriod):
			s.sendOnce(*chatID, uid, "cycle_h", "🩸 <b>Hari pertama mens</b> — hangatkan tubuh & minum cukup ya.", now)

		// (Opsional) H-2 kalau masih ingin tetap ada
		case sameDay(now, nextPeriod.AddDate(0, 0, -2)):
			s.sendOnce(*chatID, uid, "cycle_h-2", "🩸 <b>Mens 2 hari lagi</b>. Siapkan keperluanmu ya.", now)
		}

		// Otomatis geser window jika sudah lewat (agar prediksi bergeser maju)
		if now.After(nextPeriod) {
			_, _ = s.DB.ExecContext(ctx, `
				UPDATE cycles 
				SET last_period_start=$1, updated_at=NOW() 
				WHERE user_id=$2`,
				nextPeriod, uid)
		}
	}
}

// ---------- Util ----------
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
		fertileStart.Format("02 Jan"), fertileEnd.Format("02 Jan"),
		ovulation.Format("02 Jan"), nextPeriod.Format("02 Jan"))
}
