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
	ErrQAEntryNotFound = errors.New("qa entry not found")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrQAEntryArchived = errors.New("qa entry is archived")
)

// QAServiceInterface defines the interface for Q&A business logic.
type QAServiceInterface interface {
	CreateEntry(sessionCode, authorUID, entryType, body string) (*models.QAEntry, error)
	ListEntries(ctx context.Context, sessionCode, cursor string, limit int) ([]models.QAEntry, string, error)
	ModerateEntry(sessionCode string, entryID, hostID uuid.UUID, status string, isHidden *bool) (*models.QAEntry, error)
	GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error)
	GetSessionByCode(code string) (*models.Session, error)
}

// QAService provides business logic for Q&A operations.
type QAService struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewQAService creates a new QAService.
func NewQAService(database *gorm.DB, redisClient *redis.Client) *QAService {
	return &QAService{
		db:  database,
		rdb: redisClient,
	}
}

// Verify that QAService implements QAServiceInterface
var _ QAServiceInterface = (*QAService)(nil)

// CreateEntry creates a new Q&A entry (question or comment).
func (s *QAService) CreateEntry(sessionCode, authorUID, entryType, body string) (*models.QAEntry, error) {
	// Validate entry type
	if entryType != "question" && entryType != "comment" {
		return nil, fmt.Errorf("%w: entry_type must be 'question' or 'comment'", ErrInvalidRequest)
	}

	// Validate body
	if body == "" {
		return nil, fmt.Errorf("%w: body cannot be empty", ErrInvalidRequest)
	}
	if len(body) > 2000 {
		return nil, fmt.Errorf("%w: body cannot exceed 2000 characters", ErrInvalidRequest)
	}

	// Find session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	// Check if session is archived
	if session.Status == "archived" {
		return nil, fmt.Errorf("%w: cannot submit Q&A to an archived session", ErrInvalidRequest)
	}

	// Create entry
	entry := &models.QAEntry{
		SessionID: session.ID,
		AuthorUID: authorUID,
		EntryType: entryType,
		Body:      body,
		Score:     0,
		Status:    "visible",
		IsHidden:  false,
	}

	if err := s.db.Create(entry).Error; err != nil {
		return nil, fmt.Errorf("create entry: %w", err)
	}

	return entry, nil
}

// ListEntries returns paginated Q&A entries for a session.
// Entries are sorted by score DESC, then created_at ASC.
// Hidden entries are filtered out.
func (s *QAService) ListEntries(ctx context.Context, sessionCode, cursor string, limit int) ([]models.QAEntry, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Find session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrSessionNotFound
		}
		return nil, "", fmt.Errorf("find session: %w", err)
	}

	// Build query
	query := s.db.Where("session_id = ? AND is_hidden = ?", session.ID, false)

	// Apply cursor pagination
	if cursor != "" {
		var cursorEntry models.QAEntry
		if err := s.db.Where("id = ?", cursor).First(&cursorEntry).Error; err == nil {
			// Cursor format: "score:created_at:id"
			// We want entries after the cursor (lower score or same score but later created_at)
			query = query.Where(
				"(score < ?) OR (score = ? AND created_at > ?) OR (score = ? AND created_at = ? AND id > ?)",
				cursorEntry.Score, cursorEntry.Score, cursorEntry.CreatedAt,
				cursorEntry.Score, cursorEntry.CreatedAt, cursorEntry.ID,
			)
		}
	}

	// Order by score DESC, then created_at ASC
	query = query.Order("score DESC, created_at ASC, id ASC")

	// Fetch entries
	var entries []models.QAEntry
	if err := query.Limit(limit + 1).Find(&entries).Error; err != nil {
		return nil, "", fmt.Errorf("fetch entries: %w", err)
	}

	// Determine next cursor
	nextCursor := ""
	if len(entries) > limit {
		nextCursor = entries[limit].ID.String()
		entries = entries[:limit]
	}

	return entries, nextCursor, nil
}

// ModerateEntry allows host to moderate a Q&A entry (pin, answer, hide, unhide).
func (s *QAService) ModerateEntry(sessionCode string, entryID, hostID uuid.UUID, status string, isHidden *bool) (*models.QAEntry, error) {
	// Find session
	var session models.Session
	if err := s.db.Where("code = ?", sessionCode).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	// Verify host ownership
	if session.HostID == nil || *session.HostID != hostID {
		return nil, ErrNotSessionHost
	}

	// Find entry
	var entry models.QAEntry
	if err := s.db.Where("id = ? AND session_id = ?", entryID, session.ID).First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQAEntryNotFound
		}
		return nil, fmt.Errorf("find entry: %w", err)
	}

	// Validate and update status if provided
	if status != "" {
		validStatuses := map[string]bool{
			"visible":  true,
			"answered": true,
			"pinned":   true,
			"archived": true,
		}
		if !validStatuses[status] {
			return nil, fmt.Errorf("%w: invalid status '%s'", ErrInvalidRequest, status)
		}
		entry.Status = status
	}

	// Update is_hidden if provided
	if isHidden != nil {
		entry.IsHidden = *isHidden
	}

	// Save changes
	if err := s.db.Save(&entry).Error; err != nil {
		return nil, fmt.Errorf("update entry: %w", err)
	}

	return &entry, nil
}

// GetEntry retrieves a single Q&A entry by ID.
func (s *QAService) GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error) {
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

// GetSessionByCode returns the session for host ownership checks.
func (s *QAService) GetSessionByCode(code string) (*models.Session, error) {
	var session models.Session
	if err := s.db.Where("code = ?", code).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}
	return &session, nil
}
