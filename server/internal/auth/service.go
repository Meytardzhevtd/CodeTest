package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/meytardzhevtd/CodeTest/pkg/storage"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrUsernameTaken      = errors.New("username already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAvatar      = errors.New("invalid avatar")
)

// MaxAvatarBytes caps the size of an uploaded avatar image.
const MaxAvatarBytes = 5 << 20 // 5 MiB

var allowedAvatarContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

type Service struct {
	repo            *Repository
	avatars         storage.AvatarStore
	accessSecret    string
	refreshSecret   string
	accessTTL       time.Duration
	refreshTTL      time.Duration
	issuer          string
	accessTokenTTL  int64
	refreshTokenTTL int64
}

func NewService(repo *Repository, avatars storage.AvatarStore) *Service {
	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if minutes, err := strconv.Atoi(v); err == nil {
			accessTTL = time.Duration(minutes) * time.Minute
		}
	}
	if v := os.Getenv("JWT_REFRESH_TTL_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil {
			refreshTTL = time.Duration(days) * 24 * time.Hour
		}
	}
	return &Service{
		repo:            repo,
		avatars:         avatars,
		accessSecret:    os.Getenv("JWT_ACCESS_SECRET"),
		refreshSecret:   os.Getenv("JWT_REFRESH_SECRET"),
		accessTTL:       accessTTL,
		refreshTTL:      refreshTTL,
		issuer:          os.Getenv("JWT_ISSUER"),
		accessTokenTTL:  int64(accessTTL.Seconds()),
		refreshTokenTTL: int64(refreshTTL.Seconds()),
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, string, error) {
	if err := validateEmail(req.Email); err != nil {
		return AuthResponse{}, "", err
	}
	if err := validatePassword(req.Password); err != nil {
		return AuthResponse{}, "", err
	}
	if err := validateUsername(req.Username); err != nil {
		return AuthResponse{}, "", err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResponse{}, "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, strings.ToLower(strings.TrimSpace(req.Email)), strings.ToLower(strings.TrimSpace(req.Username)), string(passwordHash))
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return AuthResponse{}, "", ErrEmailTaken
		}
		if errors.Is(err, ErrUsernameTaken) {
			return AuthResponse{}, "", ErrUsernameTaken
		}
		return AuthResponse{}, "", err
	}

	return s.issueResponse(user)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, string, error) {
	if err := validateEmail(req.Email); err != nil {
		return AuthResponse{}, "", err
	}
	if err := validatePassword(req.Password); err != nil {
		return AuthResponse{}, "", err
	}

	user, passwordHash, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return AuthResponse{}, "", ErrInvalidCredentials
		}
		return AuthResponse{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return AuthResponse{}, "", ErrInvalidCredentials
	}

	return s.issueResponse(user)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenResponse, error) {
	claims, err := s.parseToken(refreshToken, s.refreshSecret)
	if err != nil {
		return TokenResponse{}, err
	}
	userID := claims.Subject
	if userID == "" {
		return TokenResponse{}, errors.New("invalid refresh token")
	}

	accessToken, err := s.signToken(userID, s.accessTTL, s.accessSecret)
	if err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{AccessToken: accessToken, TokenType: "Bearer", ExpiresIn: s.accessTokenTTL}, nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	if strings.TrimSpace(userID) == "" {
		return User{}, ErrInvalidCredentials
	}
	user, _, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	return user, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (User, error) {
	if strings.TrimSpace(userID) == "" {
		return User{}, ErrInvalidCredentials
	}

	var username string
	if req.Username != "" {
		username = strings.ToLower(strings.TrimSpace(req.Username))
		if err := validateUsername(username); err != nil {
			return User{}, err
		}
	}

	var passwordHash string
	if req.CurrentPassword != "" || req.NewPassword != "" {
		if req.CurrentPassword == "" || req.NewPassword == "" {
			return User{}, errors.New("current password and new password are required")
		}
		if err := validatePassword(req.NewPassword); err != nil {
			return User{}, err
		}

		_, currentHash, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return User{}, err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
			return User{}, ErrInvalidCredentials
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return User{}, fmt.Errorf("hash password: %w", err)
		}
		passwordHash = string(hashed)
	}

	if username == "" && passwordHash == "" {
		return User{}, errors.New("nothing to update")
	}

	user, err := s.repo.UpdateUserProfile(ctx, userID, username, passwordHash)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			return User{}, ErrUsernameTaken
		}
		return User{}, err
	}
	return user, nil
}

func (s *Service) UploadAvatar(ctx context.Context, userID string, data io.Reader, size int64, contentType string) (User, error) {
	if strings.TrimSpace(userID) == "" {
		return User{}, ErrInvalidCredentials
	}
	if size <= 0 || size > MaxAvatarBytes {
		return User{}, fmt.Errorf("%w: avatar size must be between 1 and %d bytes", ErrInvalidAvatar, MaxAvatarBytes)
	}
	if !allowedAvatarContentTypes[contentType] {
		return User{}, fmt.Errorf("%w: unsupported content type %q, expected JPEG, PNG or WebP", ErrInvalidAvatar, contentType)
	}

	key, err := s.avatars.UploadAvatar(ctx, userID, data, size, contentType)
	if err != nil {
		return User{}, err
	}

	// Cache-bust: the object is always stored at the same key, so without a
	// changing query param browsers keep serving the previous avatar.
	avatarURL := fmt.Sprintf("%s?v=%d", s.avatars.PublicURL(key), time.Now().UnixNano())

	user, err := s.repo.UpdateUserAvatar(ctx, userID, avatarURL)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) issueResponse(user User) (AuthResponse, string, error) {
	accessToken, err := s.signToken(user.ID, s.accessTTL, s.accessSecret)
	if err != nil {
		return AuthResponse{}, "", err
	}
	refreshToken, err := s.signToken(user.ID, s.refreshTTL, s.refreshSecret)
	if err != nil {
		return AuthResponse{}, "", err
	}

	return AuthResponse{
		TokenResponse: TokenResponse{AccessToken: accessToken, TokenType: "Bearer", ExpiresIn: s.accessTokenTTL},
		User:          user,
	}, refreshToken, nil
}

func (s *Service) signToken(subject string, ttl time.Duration, secret string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    s.issuer,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *Service) parseToken(tokenString, secret string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return errors.New("invalid email")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func validateUsername(username string) error {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" || len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	return nil
}
