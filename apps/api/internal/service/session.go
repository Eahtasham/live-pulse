package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
)

const (
	codeCharset    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength     = 6
	maxCodeRetries = 10
	audienceTTL    = 24 * time.Hour
)

type SessionService struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewSessionService(db *gorm.DB, rdb *redis.Client) *SessionService {
	return &SessionService{db: db, rdb: rdb}
}

// CreateSession creates a new session with a unique 6-char code.
func (s *SessionService) CreateSession(hostID uuid.UUID, title string) (*models.Session, error) {
	code, err := s.generateUniqueCode()
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	session := models.Session{
		HostID: &hostID,
		Code:   code,
		Title:  title,
		Status: "active",
	}

	if err := s.db.Create(&session).Error; err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &session, nil
}

// ListSessionsByHost returns all sessions for a given host, sorted by created_at DESC.
func (s *SessionService) ListSessionsByHost(hostID uuid.UUID) ([]models.Session, error) {
	var sessions []models.Session
	if err := s.db.Where("host_id = ?", hostID).Order("created_at DESC").Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return sessions, nil
}

// GetSessionByCode returns a session by its 6-char code.
func (s *SessionService) GetSessionByCode(code string) (*models.Session, error) {
	var session models.Session
	if err := s.db.Where("code = ?", code).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// JoinSession returns an existing or new audience UID for a client+session pair.
func (s *SessionService) JoinSession(ctx context.Context, code, clientID string) (string, *models.Session, error) {
	session, err := s.GetSessionByCode(code)
	if err != nil {
		return "", nil, err
	}

	// If client ID provided, check for existing UID
	if clientID != "" {
		redisKey := fmt.Sprintf("audience:%s:client:%s", code, clientID)
		existing, err := s.rdb.Get(ctx, redisKey).Result()
		if err == nil && existing != "" {
			return existing, session, nil
		}
	}

	// Generate new UID
	uid := uuid.New().String()

	// Store audience:{code}:{uid} with TTL
	uidKey := fmt.Sprintf("audience:%s:%s", code, uid)
	if err := s.rdb.Set(ctx, uidKey, "1", audienceTTL).Err(); err != nil {
		return "", nil, fmt.Errorf("store audience uid: %w", err)
	}

	// If client ID provided, map client→uid for idempotency
	if clientID != "" {
		clientKey := fmt.Sprintf("audience:%s:client:%s", code, clientID)
		if err := s.rdb.Set(ctx, clientKey, uid, audienceTTL).Err(); err != nil {
			return "", nil, fmt.Errorf("store client mapping: %w", err)
		}
	}

	return uid, session, nil
}

func (s *SessionService) generateUniqueCode() (string, error) {
	charsetLen := big.NewInt(int64(len(codeCharset)))

	for attempt := 0; attempt < maxCodeRetries; attempt++ {
		code := make([]byte, codeLength)
		for i := range code {
			idx, err := rand.Int(rand.Reader, charsetLen)
			if err != nil {
				return "", fmt.Errorf("crypto rand: %w", err)
			}
			code[i] = codeCharset[idx.Int64()]
		}

		candidate := string(code)

		// Check DB uniqueness
		var count int64
		if err := s.db.Model(&models.Session{}).Where("code = ?", candidate).Count(&count).Error; err != nil {
			return "", fmt.Errorf("check code uniqueness: %w", err)
		}
		if count == 0 {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxCodeRetries)
}
