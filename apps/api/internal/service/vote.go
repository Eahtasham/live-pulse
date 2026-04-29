package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
)

var (
	ErrPollNotActive      = errors.New("poll is not active")
	ErrPollClosed         = errors.New("poll is closed")
	ErrInvalidOption      = errors.New("invalid option for this poll")
	ErrDuplicateVote      = errors.New("already voted on this poll")
	ErrInvalidAudienceUID = errors.New("invalid audience uid")
	ErrSingleModeMultiple = errors.New("single mode polls only allow one option")
	ErrNoOptions          = errors.New("no options provided")
)

type VoteService struct {
	db  *gorm.DB
	rdb *redis.Client
	pub *Publisher
}

func NewVoteService(db *gorm.DB, rdb *redis.Client, pub *Publisher) *VoteService {
	return &VoteService{db: db, rdb: rdb, pub: pub}
}

// CastVote handles voting on a poll with business logic validation.
func (s *VoteService) CastVote(ctx context.Context, sessionCode string, pollID uuid.UUID, optionIDs []uuid.UUID, audienceUID string) error {
	// Validate audience UID exists in Redis
	uidKey := fmt.Sprintf("audience:%s:%s", sessionCode, audienceUID)
	exists, err := s.rdb.Exists(ctx, uidKey).Result()
	if err != nil {
		return fmt.Errorf("redis check: %w", err)
	}
	if exists == 0 {
		return ErrInvalidAudienceUID
	}

	// Validate at least one option
	if len(optionIDs) == 0 {
		return ErrNoOptions
	}

	// Get session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("find session: %w", err)
	}

	if session.Status == "archived" {
		return ErrSessionArchived
	}

	// Get poll with options
	var poll models.Poll
	if err := s.db.Preload("Options").Where("id = ? AND session_id = ?", pollID, session.ID).First(&poll).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPollNotFound
		}
		return fmt.Errorf("find poll: %w", err)
	}

	// Check poll status
	if poll.Status == "draft" {
		return ErrPollNotActive
	}
	if poll.Status == "closed" {
		return ErrPollClosed
	}
	if poll.Status != "active" {
		return ErrPollNotActive
	}

	// Validate answer mode
	if poll.AnswerMode == "single" && len(optionIDs) > 1 {
		return ErrSingleModeMultiple
	}

	// Build valid option ID set
	validOptionIDs := make(map[uuid.UUID]bool)
	for _, opt := range poll.Options {
		validOptionIDs[opt.ID] = true
	}

	// Validate all option IDs belong to this poll
	for _, optID := range optionIDs {
		if !validOptionIDs[optID] {
			return ErrInvalidOption
		}
	}

	// Check for existing votes by this audience on this poll
	var existingCount int64
	if err := s.db.Model(&models.Vote{}).
		Where("poll_id = ? AND audience_uid = ?", pollID, audienceUID).
		Count(&existingCount).Error; err != nil {
		return fmt.Errorf("check existing votes: %w", err)
	}
	if existingCount > 0 {
		return ErrDuplicateVote
	}

	// Create vote records
	for _, optID := range optionIDs {
		vote := models.Vote{
			PollID:      pollID,
			OptionID:    optID,
			AudienceUID: audienceUID,
		}
		if err := s.db.Create(&vote).Error; err != nil {
			return fmt.Errorf("create vote: %w", err)
		}
	}

	// Publish vote_update with full poll state
	if s.pub != nil {
		go func() {
			counts, err := s.GetVoteCounts(pollID)
			if err != nil {
				return
			}
			var optPayloads []VoteOptionPayload
			for _, opt := range poll.Options {
				optPayloads = append(optPayloads, VoteOptionPayload{
					ID:        opt.ID.String(),
					Label:     opt.Label,
					VoteCount: counts[opt.ID],
				})
			}
			s.pub.PublishVoteUpdate(context.Background(), sessionCode, pollID, optPayloads)
		}()
	}

	return nil
}

// GetVoteCounts returns vote counts per option for a poll.
func (s *VoteService) GetVoteCounts(pollID uuid.UUID) (map[uuid.UUID]int64, error) {
	type voteCount struct {
		OptionID uuid.UUID `gorm:"column:option_id"`
		Count    int64     `gorm:"column:count"`
	}

	var counts []voteCount
	if err := s.db.Model(&models.Vote{}).
		Select("option_id, COUNT(*) as count").
		Where("poll_id = ?", pollID).
		Group("option_id").
		Find(&counts).Error; err != nil {
		return nil, fmt.Errorf("count votes: %w", err)
	}

	voteCounts := make(map[uuid.UUID]int64)
	for _, c := range counts {
		voteCounts[c.OptionID] = c.Count
	}

	return voteCounts, nil
}

// MyVoteEntry represents a single vote record for the audience member.
type MyVoteEntry struct {
	PollID    string   `json:"poll_id"`
	OptionIDs []string `json:"option_ids"`
}

// GetMyVotes returns all votes cast by the given audience UID within a session.
func (s *VoteService) GetMyVotes(sessionCode, audienceUID string) ([]MyVoteEntry, error) {
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	var votes []models.Vote
	if err := s.db.
		Joins("JOIN polls ON polls.id = votes.poll_id").
		Where("polls.session_id = ? AND votes.audience_uid = ?", session.ID, audienceUID).
		Find(&votes).Error; err != nil {
		return nil, fmt.Errorf("find votes: %w", err)
	}

	// Group by poll_id
	grouped := make(map[string][]string)
	for _, v := range votes {
		pid := v.PollID.String()
		grouped[pid] = append(grouped[pid], v.OptionID.String())
	}

	result := make([]MyVoteEntry, 0, len(grouped))
	for pid, oids := range grouped {
		result = append(result, MyVoteEntry{PollID: pid, OptionIDs: oids})
	}

	return result, nil
}
