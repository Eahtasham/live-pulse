package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
)

var (
	ErrQAEntryIsComment    = errors.New("cannot vote on comments")
	ErrQAEntryNotVisible   = errors.New("qa entry is not visible")
	ErrInvalidVoteValue    = errors.New("vote value must be 1 (upvote) or -1 (downvote)")
)

// QAVoteServiceInterface defines the interface for Q&A vote business logic.
type QAVoteServiceInterface interface {
	CastVote(sessionCode string, entryID uuid.UUID, voterUID string, value int16) (*models.QAVote, error)
	GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error)
}

// QAVoteService provides business logic for Q&A voting operations.
type QAVoteService struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewQAVoteService creates a new QAVoteService.
func NewQAVoteService(database *gorm.DB, redisClient *redis.Client) *QAVoteService {
	return &QAVoteService{
		db:  database,
		rdb: redisClient,
	}
}

// Verify that QAVoteService implements QAVoteServiceInterface
var _ QAVoteServiceInterface = (*QAVoteService)(nil)

// CastVote casts an upvote or downvote on a Q&A entry.
// Implements toggle behavior: upvoting twice removes the vote.
// Changing from upvote to downvote updates the existing vote.
func (s *QAVoteService) CastVote(sessionCode string, entryID uuid.UUID, voterUID string, value int16) (*models.QAVote, error) {
	// Validate vote value
	if value != 1 && value != -1 {
		return nil, ErrInvalidVoteValue
	}

	// Find session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	// Find entry
	var entry models.QAEntry
	if err := s.db.Where("id = ? AND session_id = ?", entryID, session.ID).First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQAEntryNotFound
		}
		return nil, fmt.Errorf("find entry: %w", err)
	}

	// Cannot vote on comments (only questions)
	if entry.EntryType == "comment" {
		return nil, ErrQAEntryIsComment
	}

	// Cannot vote on hidden or archived entries
	if entry.IsHidden {
		return nil, ErrQAEntryNotVisible
	}
	if entry.Status == "archived" {
		return nil, ErrQAEntryArchived
	}

	// Check for existing vote
	var existingVote models.QAVote
	err := s.db.Where("qa_entry_id = ? AND voter_uid = ?", entryID, voterUID).First(&existingVote).Error

	if err == nil {
		// Existing vote found - toggle behavior
		if existingVote.VoteValue == int16(value) {
			// Same vote value - remove the vote (toggle off)
			if err := s.db.Delete(&existingVote).Error; err != nil {
				return nil, fmt.Errorf("remove vote: %w", err)
			}
			// Recalculate score
			if err := s.recalculateScore(entryID); err != nil {
				return nil, fmt.Errorf("recalculate score: %w", err)
			}
			return nil, nil // Vote removed
		} else {
			// Different vote value - update the vote
			existingVote.VoteValue = int16(value)
			if err := s.db.Save(&existingVote).Error; err != nil {
				return nil, fmt.Errorf("update vote: %w", err)
			}
			// Recalculate score
			if err := s.recalculateScore(entryID); err != nil {
				return nil, fmt.Errorf("recalculate score: %w", err)
			}
			return &existingVote, nil
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing vote: %w", err)
	}

	// No existing vote - create new vote
	vote := &models.QAVote{
		QAEntryID: entryID,
		VoterUID:  voterUID,
		VoteValue: int16(value),
	}

	if err := s.db.Create(vote).Error; err != nil {
		return nil, fmt.Errorf("create vote: %w", err)
	}

	// Recalculate score
	if err := s.recalculateScore(entryID); err != nil {
		return nil, fmt.Errorf("recalculate score: %w", err)
	}

	return vote, nil
}

// recalculateScore recalculates the score for a Q&A entry based on all votes.
func (s *QAVoteService) recalculateScore(entryID uuid.UUID) error {
	var totalScore int64
	if err := s.db.Model(&models.QAVote{}).
		Where("qa_entry_id = ?", entryID).
		Select("COALESCE(SUM(vote_value), 0)").
		Scan(&totalScore).Error; err != nil {
		return fmt.Errorf("sum votes: %w", err)
	}

	if err := s.db.Model(&models.QAEntry{}).
		Where("id = ?", entryID).
		Update("score", totalScore).Error; err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	return nil
}

// GetEntry retrieves a single Q&A entry by ID (for handler layer access).
func (s *QAVoteService) GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error) {
	// Find session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	// Find entry
	var entry models.QAEntry
	if err := s.db.Where("id = ? AND session_id = ?", entryID, session.ID).First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQAEntryNotFound
		}
		return nil, fmt.Errorf("find entry: %w", err)
	}

	return &entry, nil
}
