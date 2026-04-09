package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
