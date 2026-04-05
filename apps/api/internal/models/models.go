package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Name      *string   `json:"name"`
	AvatarURL *string   `json:"avatar_url"`
	Provider  string    `gorm:"not null" json:"provider"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

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

type QAEntry struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	AuthorUID string    `gorm:"not null" json:"author_uid"`
	EntryType string    `gorm:"not null" json:"entry_type"`
	Body      string    `gorm:"not null" json:"body"`
	Score     int       `gorm:"not null;default:0" json:"score"`
	Status    string    `gorm:"not null;default:visible" json:"status"`
	IsHidden  bool      `gorm:"not null;default:false" json:"is_hidden"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (QAEntry) TableName() string { return "qa_entries" }

type QAVote struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QAEntryID uuid.UUID `gorm:"type:uuid;not null;index" json:"qa_entry_id"`
	VoterUID  string    `gorm:"not null" json:"voter_uid"`
	VoteValue int16     `gorm:"not null" json:"vote_value"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (QAVote) TableName() string { return "qa_votes" }

// BeforeCreate hook to generate UUID if not set
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

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

func (q *QAEntry) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

func (qv *QAVote) BeforeCreate(tx *gorm.DB) error {
	if qv.ID == uuid.Nil {
		qv.ID = uuid.New()
	}
	return nil
}
