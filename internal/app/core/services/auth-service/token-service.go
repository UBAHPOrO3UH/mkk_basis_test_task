package auth_service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mkk_basis/rest_api/internal/config"
	"strconv"
	"strings"
	"time"
)

const (
	AccessTokenType  = "access"
	RefreshTokenType = "refresh"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

type Claims struct {
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	ID        string `json:"jti"`
}

func (c *Claims) UserID() (uint64, error) {
	id, err := strconv.ParseUint(c.Subject, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidToken
	}
	return id, nil
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type AccessToken struct {
	Token     string
	ExpiresAt time.Time
	Claims    *Claims
}

type TokenService struct {
	config *config.AuthConfig
	now    func() time.Time
}

func NewTokenService(cfg *config.AuthConfig) *TokenService {
	return &TokenService{config: cfg, now: time.Now}
}

func (s *TokenService) IssuePair(userID uint64, username string) (*TokenPair, error) {
	now := s.now().UTC()
	accessExpiresAt := now.Add(time.Duration(s.config.AccessTokenTTLMinutes) * time.Minute)
	refreshExpiresAt := now.Add(time.Duration(s.config.RefreshTokenTTLHours) * time.Hour)

	accessToken, err := s.issue(userID, username, AccessTokenType, now, accessExpiresAt)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.issue(userID, username, RefreshTokenType, now, refreshExpiresAt)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *TokenService) IssueAccess(userID uint64, username string) (*AccessToken, error) {
	now := s.now().UTC()
	expiresAt := now.Add(time.Duration(s.config.AccessTokenTTLMinutes) * time.Minute)
	token, err := s.issue(userID, username, AccessTokenType, now, expiresAt)
	if err != nil {
		return nil, err
	}

	claims, err := s.ParseAccess(token)
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: token, ExpiresAt: expiresAt, Claims: claims}, nil
}

func (s *TokenService) ParseAccess(token string) (*Claims, error) {
	return s.parse(token, AccessTokenType)
}

func (s *TokenService) ParseRefresh(token string) (*Claims, error) {
	return s.parse(token, RefreshTokenType)
}

func (s *TokenService) issue(
	userID uint64,
	username string,
	tokenType string,
	issuedAt time.Time,
	expiresAt time.Time,
) (string, error) {
	if userID == 0 || strings.TrimSpace(username) == "" {
		return "", ErrInvalidToken
	}

	headerJSON, err := json.Marshal(struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", err
	}

	jtiBytes := make([]byte, 16)
	if _, err = rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}

	claimsJSON, err := json.Marshal(&Claims{
		Subject:   strconv.FormatUint(userID, 10),
		Username:  username,
		TokenType: tokenType,
		Issuer:    s.config.JWTIssuer,
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
		ID:        hex.EncodeToString(jtiBytes),
	})
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	return unsigned + "." + s.signature(unsigned), nil
}

func (s *TokenService) parse(token, expectedType string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err = json.Unmarshal(headerJSON, &header); err != nil ||
		header.Algorithm != "HS256" || header.Type != "JWT" {
		return nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	expectedSignature, _ := base64.RawURLEncoding.DecodeString(s.signature(unsigned))
	if !hmac.Equal(actualSignature, expectedSignature) {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err = json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType ||
		claims.Issuer != s.config.JWTIssuer ||
		claims.Username == "" ||
		claims.ID == "" ||
		claims.ExpiresAt <= claims.IssuedAt {
		return nil, ErrInvalidToken
	}
	if _, err = claims.UserID(); err != nil {
		return nil, ErrInvalidToken
	}

	now := s.now().UTC().Unix()
	if now >= claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	if claims.IssuedAt > now+30 {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

func (s *TokenService) signature(value string) string {
	mac := hmac.New(sha256.New, []byte(s.config.JWTSecret))
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
