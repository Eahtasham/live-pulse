package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Session struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	HostID    *uuid.UUID `gorm:"type:uuid;index" json:"host_id"`
	Code      string     `gorm:"type:varchar(6);uniqueIndex;not null" json:"code"`
	Title     string     `gorm:"not null" json:"title"`
	Status    string     `gorm:"not null;default:active" json:"status"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`

	Host  *User  `gorm:"foreignKey:HostID;constraint:OnDelete:SET NULL" json:"host,omitempty"`
	Polls []Poll `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"polls,omitempty"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
