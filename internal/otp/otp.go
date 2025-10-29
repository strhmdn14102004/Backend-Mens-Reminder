package otp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"
)

func Generate(n int) (string, error) {
	// numeric OTP
	const digits = "0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[int(b[i])%10]
	}
	return string(b), nil
}

func CreateOTP(ctx context.Context, db *sql.DB, phone, code string, ttl time.Duration) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO otps (phone, code, expires_at) VALUES ($1,$2,$3)`,
		phone, code, time.Now().Add(ttl))
	return err
}

func VerifyOTP(ctx context.Context, db *sql.DB, phone, code string) (bool, error) {
	var id int64
	var expires time.Time
	var consumed bool
	err := db.QueryRowContext(ctx, `SELECT id, expires_at, consumed FROM otps WHERE phone=$1 AND code=$2 ORDER BY id DESC LIMIT 1`,
		phone, code).Scan(&id, &expires, &consumed)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if consumed || time.Now().After(expires) {
		return false, nil
	}
	_, _ = db.ExecContext(ctx, `UPDATE otps SET consumed=TRUE WHERE id=$1`, id)
	return true, nil
}

// Kirim SMS via provider eksternal.
// Di dev, bisa log ke console; di prod, ganti implementasi ini.
func SendSMS(phone, message string) error {
	fmt.Printf("[DEV SMS] to %s: %s\n", phone, message)
	return nil
}
