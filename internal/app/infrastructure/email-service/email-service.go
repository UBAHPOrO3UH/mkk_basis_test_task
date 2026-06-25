package email_service

import (
	"context"
	"errors"
	"fmt"
	"mkk_basis/rest_api/internal/config"
	"sync"
	"time"
)

var (
	ErrEmailServiceUnavailable = errors.New("email service unavailable")
	ErrCircuitBreakerOpen      = errors.New("email service circuit breaker is open")
)

type InvitationEmail struct {
	TeamID    uint64
	UserID    uint64
	Username  string
	InviterID uint64
}

type EmailService interface {
	SendInvitation(ctx context.Context, email InvitationEmail) error
}

type NoopEmailService struct{}

func NewNoopEmailService() EmailService {
	return &NoopEmailService{}
}

func (s *NoopEmailService) SendInvitation(context.Context, InvitationEmail) error {
	return nil
}

type MockEmailService struct {
	cfg     *config.EmailConfig
	breaker *CircuitBreaker
}

func NewMockEmailService(cfg *config.EmailConfig) EmailService {
	if cfg == nil || !cfg.Enabled {
		return NewNoopEmailService()
	}

	return &MockEmailService{
		cfg:     cfg,
		breaker: NewCircuitBreaker(cfg.CircuitBreaker),
	}
}

func (s *MockEmailService) SendInvitation(ctx context.Context, email InvitationEmail) error {
	if s == nil || s.cfg == nil || !s.cfg.Enabled {
		return nil
	}

	return s.breaker.Execute(func() error {
		if s.cfg.MockLatencyMS > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(s.cfg.MockLatencyMS) * time.Millisecond):
			}
		}

		if s.cfg.MockFailure {
			return fmt.Errorf("%w: mock failure for team_id=%d user_id=%d", ErrEmailServiceUnavailable, email.TeamID, email.UserID)
		}

		return nil
	})
}

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

type CircuitBreaker struct {
	mu               sync.Mutex
	state            circuitState
	failureCount     int
	failureThreshold int
	openedAt         time.Time
	openTimeout      time.Duration
}

func NewCircuitBreaker(cfg *config.CircuitBreakerConfig) *CircuitBreaker {
	failureThreshold := 3
	openTimeout := 30 * time.Second
	if cfg != nil {
		if cfg.FailureThreshold > 0 {
			failureThreshold = cfg.FailureThreshold
		}
		if cfg.OpenTimeoutSeconds > 0 {
			openTimeout = time.Duration(cfg.OpenTimeoutSeconds) * time.Second
		}
	}

	return &CircuitBreaker{
		state:            circuitClosed,
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
	}
}

func (b *CircuitBreaker) Execute(fn func() error) error {
	if err := b.beforeCall(); err != nil {
		return err
	}

	err := fn()
	b.afterCall(err)
	return err
}

func (b *CircuitBreaker) beforeCall() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != circuitOpen {
		return nil
	}

	if time.Since(b.openedAt) < b.openTimeout {
		return ErrCircuitBreakerOpen
	}

	b.state = circuitHalfOpen
	return nil
}

func (b *CircuitBreaker) afterCall(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err == nil {
		b.state = circuitClosed
		b.failureCount = 0
		b.openedAt = time.Time{}
		return
	}

	b.failureCount++
	if b.state == circuitHalfOpen || b.failureCount >= b.failureThreshold {
		b.state = circuitOpen
		b.openedAt = time.Now()
	}
}
