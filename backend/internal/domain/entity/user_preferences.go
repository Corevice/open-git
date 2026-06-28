package entity

import "time"

type UserPreferences struct {
	UserID    int64
	Theme     string
	UpdatedAt time.Time
}
