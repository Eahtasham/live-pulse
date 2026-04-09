package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Poll struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID    uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	Question     string    `gorm:"not null" json:"question"`
	AnswerMode   string    `gorm:"not null;default:single" json:"answer_mode"`
	TimeLimitSec *int      `json:"time_limit_sec"`
	Status       string    `gorm:"not null;default:draft" json:"status"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Options []PollOption `gorm:"foreignKey:PollID;constraint:OnDelete:CASCADE" json:"options,omitempty"`
}

type PollOption struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PollID   uuid.UUID `gorm:"type:uuid;not null;index" json:"poll_id"`
	Label    string    `gorm:"not null" json:"label"`
	Position int16     `gorm:"not null" json:"position"`
}

type Vote struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PollID      uuid.UUID `gorm:"type:uuid;not null;index" json:"poll_id"`
	OptionID    uuid.UUID `gorm:"type:uuid;not null;index" json:"option_id"`
	AudienceUID string    `gorm:"not null" json:"audience_uid"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (Vote) TableName() string { return "votes" }

func (p *Poll) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (po *PollOption) BeforeCreate(tx *gorm.DB) error {
	if po.ID == uuid.Nil {
		po.ID = uuid.New()
	}
	return nil
}

func (v *Vote) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
