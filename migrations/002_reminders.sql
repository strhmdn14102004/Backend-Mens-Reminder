-- Log pengiriman (anti duplikasi)
CREATE TABLE IF NOT EXISTS reminder_logs (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind       TEXT   NOT NULL,   -- contoh: 'cycle_h-3','cycle_h-1','cycle_h','ovulation','fertile_end','wellness_am','wellness_pm'
  sent_date  DATE   NOT NULL,   -- tanggal lokal (WIB) ketika dikirim
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, kind, sent_date)
);

-- Preferensi user (default semua aktif)
CREATE TABLE IF NOT EXISTS reminder_prefs (
  user_id      BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  wellness_am  BOOLEAN NOT NULL DEFAULT TRUE,
  wellness_pm  BOOLEAN NOT NULL DEFAULT TRUE,
  cycle_alerts BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
