-- 001_init.sql
CREATE TABLE users (
  id                BIGSERIAL PRIMARY KEY,
  phone             TEXT UNIQUE NOT NULL,
  email             TEXT,
  name              TEXT,
  birth_date        DATE,
  password_hash     TEXT,                -- optional kalau suatu saat mau pakai pwd
  telegram_chat_id  BIGINT,              -- null kalau belum connect
  created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE cycles (
  user_id           BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  last_period_start DATE NOT NULL,
  cycle_length      INT  NOT NULL,       -- rata2 panjang siklus, default 28
  period_length     INT  NOT NULL,       -- rata2 lama mens, default 5
  updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE otps (
  id                BIGSERIAL PRIMARY KEY,
  phone             TEXT NOT NULL,
  code              TEXT NOT NULL,
  expires_at        TIMESTAMP NOT NULL,
  consumed          BOOLEAN NOT NULL DEFAULT FALSE,
  created_at        TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otps_phone ON otps(phone);


CREATE TABLE telegram_links (
  id                BIGSERIAL PRIMARY KEY,
  user_id           BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  link_token        TEXT UNIQUE NOT NULL,  -- dipakai di /start <token>
  created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  consumed_at       TIMESTAMP
);
