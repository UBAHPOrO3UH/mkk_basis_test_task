package auth_service

import (
	"errors"
	"mkk_basis/rest_api/internal/config"
	"strings"
	"testing"
	"time"
)

func testTokenService(now time.Time) *TokenService {
	service := NewTokenService(&config.AuthConfig{
		JWTSecret:             "01234567890123456789012345678901",
		JWTIssuer:             "test",
		AccessTokenTTLMinutes: 15,
		RefreshTokenTTLHours:  24,
	})
	service.now = func() time.Time { return now }
	return service
}

func TestTokenServiceIssueAndParse(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	service := testTokenService(now)

	pair, err := service.IssuePair(42, "ivan")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	accessClaims, err := service.ParseAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if accessClaims.Subject != "42" || accessClaims.Username != "ivan" {
		t.Fatalf("unexpected access claims: %+v", accessClaims)
	}

	if _, err = service.ParseAccess(pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseAccess(refresh) error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenServiceRejectsExpiredAndTamperedToken(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	service := testTokenService(now)
	pair, err := service.IssuePair(42, "ivan")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	service.now = func() time.Time { return pair.AccessExpiresAt }
	if _, err = service.ParseAccess(pair.AccessToken); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("ParseAccess(expired) error = %v, want ErrTokenExpired", err)
	}

	parts := strings.Split(pair.RefreshToken, ".")
	parts[1] = parts[1][:len(parts[1])-1] + "A"
	if _, err = service.ParseRefresh(strings.Join(parts, ".")); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseRefresh(tampered) error = %v, want ErrInvalidToken", err)
	}
}
