package models

import "time"

type User struct {
	ID              int64
	Phone           string
	Email           *string
	Name            *string
	BirthDate       *time.Time
	TelegramChatID  *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Cycle struct {
	UserID          int64
	LastPeriodStart time.Time
	CycleLength     int
	PeriodLength    int
	UpdatedAt       time.Time
}
