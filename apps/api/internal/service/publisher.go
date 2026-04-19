package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Event represents a real-time event published to Redis.
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// VoteUpdatePayload is sent when a vote is cast, containing ALL options with current counts.
type VoteUpdatePayload struct {
	PollID  string              `json:"pollId"`
	Options []VoteOptionPayload `json:"options"`
}

type VoteOptionPayload struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	VoteCount int64  `json:"vote_count"`
}

// NewQuestionPayload is sent when a new question is submitted.
type NewQuestionPayload struct {
	ID        string `json:"id"`
	EntryType string `json:"entry_type"`
	Body      string `json:"body"`
	Score     int    `json:"score"`
	AuthorUID string `json:"author_uid"`
	CreatedAt string `json:"created_at"`
}

// NewCommentPayload is sent when a new comment is submitted.
type NewCommentPayload struct {
	ID        string `json:"id"`
	EntryType string `json:"entry_type"`
	Body      string `json:"body"`
	AuthorUID string `json:"author_uid"`
	CreatedAt string `json:"created_at"`
}

// QAUpdatePayload is sent on moderation actions or Q&A votes.
type QAUpdatePayload struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	IsHidden bool   `json:"is_hidden"`
	Score    int    `json:"score"`
}

// SessionClosedPayload is sent when a session is closed.
type SessionClosedPayload struct {
	Code     string `json:"code"`
	ClosedAt string `json:"closed_at"`
}

// Publisher wraps Redis PUBLISH with typed event constructors.
type Publisher struct {
	rdb *redis.Client
}

// NewPublisher creates a new Publisher.
func NewPublisher(rdb *redis.Client) *Publisher {
	return &Publisher{rdb: rdb}
}

func (p *Publisher) publish(ctx context.Context, code string, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal event", "type", event.Type, "error", err)
		return
	}

	channel := "session:" + code
	if err := p.rdb.Publish(ctx, channel, data).Err(); err != nil {
		slog.Error("failed to publish event", "channel", channel, "type", event.Type, "error", err)
	} else {
		slog.Debug("published event", "channel", channel, "type", event.Type)
	}
}

// PublishVoteUpdate publishes a vote_update event with full poll state.
func (p *Publisher) PublishVoteUpdate(ctx context.Context, code string, pollID uuid.UUID, options []VoteOptionPayload) {
	p.publish(ctx, code, Event{
		Type: "vote_update",
		Payload: VoteUpdatePayload{
			PollID:  pollID.String(),
			Options: options,
		},
	})
}

// PublishNewQuestion publishes a new_question event.
func (p *Publisher) PublishNewQuestion(ctx context.Context, code string, id uuid.UUID, body, authorUID string, score int, createdAt time.Time) {
	p.publish(ctx, code, Event{
		Type: "new_question",
		Payload: NewQuestionPayload{
			ID:        id.String(),
			EntryType: "question",
			Body:      body,
			Score:     score,
			AuthorUID: authorUID,
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	})
}

// PublishNewComment publishes a new_comment event.
func (p *Publisher) PublishNewComment(ctx context.Context, code string, id uuid.UUID, body, authorUID string, createdAt time.Time) {
	p.publish(ctx, code, Event{
		Type: "new_comment",
		Payload: NewCommentPayload{
			ID:        id.String(),
			EntryType: "comment",
			Body:      body,
			AuthorUID: authorUID,
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	})
}

// PublishQAUpdate publishes a qa_update event.
func (p *Publisher) PublishQAUpdate(ctx context.Context, code string, id uuid.UUID, status string, isHidden bool, score int) {
	p.publish(ctx, code, Event{
		Type: "qa_update",
		Payload: QAUpdatePayload{
			ID:       id.String(),
			Status:   status,
			IsHidden: isHidden,
			Score:    score,
		},
	})
}

// PublishSessionClosed publishes a session_closed event.
func (p *Publisher) PublishSessionClosed(ctx context.Context, code string, closedAt time.Time) {
	p.publish(ctx, code, Event{
		Type: "session_closed",
		Payload: SessionClosedPayload{
			Code:     code,
			ClosedAt: closedAt.Format(time.RFC3339),
		},
	})
}

// BuildVoteOptionPayloads constructs VoteOptionPayload slice from poll options and vote counts.
func BuildVoteOptionPayloads(pollID uuid.UUID, options []struct {
	ID    uuid.UUID
	Label string
}, voteCounts map[uuid.UUID]int64) []VoteOptionPayload {
	result := make([]VoteOptionPayload, len(options))
	for i, opt := range options {
		result[i] = VoteOptionPayload{
			ID:        opt.ID.String(),
			Label:     opt.Label,
			VoteCount: voteCounts[opt.ID],
		}
	}
	return result
}

// GetPollOptionsWithCounts fetches poll options and vote counts, returning VoteOptionPayloads.
func (p *Publisher) GetPollOptionsWithCounts(ctx context.Context, pollID uuid.UUID, db interface {
	GetPollWithOptions(pollID uuid.UUID) ([]struct {
		ID    uuid.UUID
		Label string
	}, error)
	GetVoteCounts(pollID uuid.UUID) (map[uuid.UUID]int64, error)
}) ([]VoteOptionPayload, error) {
	options, err := db.GetPollWithOptions(pollID)
	if err != nil {
		return nil, fmt.Errorf("get poll options: %w", err)
	}

	counts, err := db.GetVoteCounts(pollID)
	if err != nil {
		return nil, fmt.Errorf("get vote counts: %w", err)
	}

	return BuildVoteOptionPayloads(pollID, options, counts), nil
}
