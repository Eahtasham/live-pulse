package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type Service struct {
	db        *gorm.DB
	jwtSecret []byte
	jwtExpiry time.Duration
}

func New(db *gorm.DB, jwtSecret string, jwtExpiry time.Duration) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

// FindOrCreateUser finds a user by email or creates one. Returns the user.
func (s *Service) FindOrCreateUser(email, name, avatarURL, provider string) (*models.User, error) {
	var user models.User
	result := s.db.Where(models.User{Email: email}).
		Attrs(models.User{
			Name:      strPtr(name),
			AvatarURL: strPtr(avatarURL),
			Provider:  provider,
		}).
		FirstOrCreate(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("find or create user: %w", result.Error)
	}
	return &user, nil
}

// RegisterUser creates a new user with email and hashed password.
func (s *Service) RegisterUser(email, name, password string) (*models.User, error) {
	var existing models.User
	result := s.db.Where("email = ?", email).First(&existing)
	if result.Error == nil {
		return nil, ErrEmailTaken
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing user: %w", result.Error)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	user := models.User{
		Email:        email,
		Name:         strPtr(name),
		Provider:     "email",
		PasswordHash: &hashStr,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &user, nil
}

// LoginUser verifies email and password, returns the user if valid.
func (s *Service) LoginUser(email, password string) (*models.User, error) {
	var user models.User
	result := s.db.Where("email = ? AND provider = ?", email, "email").First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", result.Error)
	}

	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

// GenerateJWT creates a signed JWT containing user_id and email.
func (s *Service) GenerateJWT(userID, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"iat":     now.Unix(),
		"exp":     now.Add(s.jwtExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
