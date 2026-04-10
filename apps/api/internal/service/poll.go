package service

import (
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
)

var (
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionArchived   = errors.New("session is archived")
	ErrNotSessionHost    = errors.New("not the session host")
	ErrPollNotFound      = errors.New("poll not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrInvalidInput      = errors.New("invalid input")
)

type PollService struct {
	db *gorm.DB
}

func NewPollService(db *gorm.DB) *PollService {
	return &PollService{db: db}
}

// CreatePoll creates a new poll with options within a session.
func (s *PollService) CreatePoll(sessionCode string, hostID uuid.UUID, question, answerMode string, timeLimitSec *int, options []models.PollOption) (*models.Poll, error) {
	// Look up session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	// Check archived
	if session.Status == "archived" {
		return nil, ErrSessionArchived
	}

	// Check host
	if session.HostID == nil || *session.HostID != hostID {
		return nil, ErrNotSessionHost
	}

	// Validate question
	if question == "" || len(question) > 500 {
		return nil, fmt.Errorf("%w: question is required and must be at most 500 characters", ErrInvalidInput)
	}

	// Validate answer_mode
	if answerMode != "single" && answerMode != "multi" {
		return nil, fmt.Errorf("%w: answer_mode must be 'single' or 'multi'", ErrInvalidInput)
	}

	// Validate options count
	if len(options) < 2 || len(options) > 6 {
		return nil, fmt.Errorf("%w: options must be between 2 and 6", ErrInvalidInput)
	}

	// Validate each option label
	for _, o := range options {
		if o.Label == "" || len(o.Label) > 200 {
			return nil, fmt.Errorf("%w: option label is required and must be at most 200 characters", ErrInvalidInput)
		}
	}

	// Validate time_limit_sec
	if timeLimitSec != nil && *timeLimitSec <= 0 {
		return nil, fmt.Errorf("%w: time_limit_sec must be a positive integer", ErrInvalidInput)
	}

	poll := models.Poll{
		SessionID:    session.ID,
		Question:     question,
		AnswerMode:   answerMode,
		TimeLimitSec: timeLimitSec,
		Status:       "draft",
		Options:      options,
	}

	if err := s.db.Create(&poll).Error; err != nil {
		return nil, fmt.Errorf("create poll: %w", err)
	}

	// Reload with options sorted by position
	if err := s.db.Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).First(&poll, "id = ?", poll.ID).Error; err != nil {
		return nil, fmt.Errorf("reload poll: %w", err)
	}

	return &poll, nil
}

// ListPolls returns polls for a session. If isHost is false, only active and closed polls are returned.
func (s *PollService) ListPolls(sessionCode string, isHost bool) ([]models.Poll, error) {
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	query := s.db.Where("session_id = ?", session.ID).Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	})

	if !isHost {
		query = query.Where("status IN ?", []string{"active", "closed"})
	}

	var polls []models.Poll
	if err := query.Order("created_at DESC").Find(&polls).Error; err != nil {
		return nil, fmt.Errorf("list polls: %w", err)
	}

	return polls, nil
}

// GetPoll returns a single poll with vote counts per option.
func (s *PollService) GetPoll(sessionCode string, pollID uuid.UUID) (*models.Poll, map[uuid.UUID]int64, error) {
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSessionNotFound
		}
		return nil, nil, fmt.Errorf("find session: %w", err)
	}

	var poll models.Poll
	if err := s.db.Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).Where("id = ? AND session_id = ?", pollID, session.ID).First(&poll).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrPollNotFound
		}
		return nil, nil, fmt.Errorf("find poll: %w", err)
	}

	// Get vote counts per option
	type voteCount struct {
		OptionID uuid.UUID `gorm:"column:option_id"`
		Count    int64     `gorm:"column:count"`
	}
	var counts []voteCount
	s.db.Model(&models.Vote{}).
		Select("option_id, COUNT(*) as count").
		Where("poll_id = ?", pollID).
		Group("option_id").
		Find(&counts)

	voteCounts := make(map[uuid.UUID]int64)
	for _, c := range counts {
		voteCounts[c.OptionID] = c.Count
	}

	return &poll, voteCounts, nil
}

// UpdatePollStatus transitions a poll's status. Only forward transitions allowed.
func (s *PollService) UpdatePollStatus(sessionCode string, pollID, hostID uuid.UUID, newStatus string) (*models.Poll, error) {
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	if session.HostID == nil || *session.HostID != hostID {
		return nil, ErrNotSessionHost
	}

	var poll models.Poll
	if err := s.db.Where("id = ? AND session_id = ?", pollID, session.ID).First(&poll).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPollNotFound
		}
		return nil, fmt.Errorf("find poll: %w", err)
	}

	// Validate transition
	if !isValidTransition(poll.Status, newStatus) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, poll.Status, newStatus)
	}

	poll.Status = newStatus
	if err := s.db.Save(&poll).Error; err != nil {
		return nil, fmt.Errorf("update poll: %w", err)
	}

	// Reload with options
	if err := s.db.Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).First(&poll, "id = ?", poll.ID).Error; err != nil {
		return nil, fmt.Errorf("reload poll: %w", err)
	}

	return &poll, nil
}

// GetSessionByCode returns the session for host ownership checks in the handler layer.
func (s *PollService) GetSessionByCode(code string) (*models.Session, error) {
	var session models.Session
	if err := s.db.Where("code = ?", code).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}
	return &session, nil
}

func isValidTransition(from, to string) bool {
	transitions := map[string]string{
		"draft":  "active",
		"active": "closed",
	}
	allowed, ok := transitions[from]
	return ok && allowed == to
}

// SortOptionsByPosition sorts options in-place by position.
func SortOptionsByPosition(options []models.PollOption) {
	sort.Slice(options, func(i, j int) bool {
		return options[i].Position < options[j].Position
	})
}
