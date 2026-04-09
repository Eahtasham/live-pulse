package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Name         *string   `json:"name"`
	AvatarURL    *string   `json:"avatar_url"`
	Provider     string    `gorm:"not null" json:"provider"`
	PasswordHash *string   `gorm:"column:password_hash" json:"-"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
