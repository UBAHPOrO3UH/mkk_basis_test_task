package middlewares

import (
	"context"
	"errors"
	auth_service "mkk_basis/rest_api/internal/app/core/services/auth-service"
	"mkk_basis/rest_api/internal/app/deps"
	"mkk_basis/rest_api/internal/common"
	"mkk_basis/rest_api/internal/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	AccessCookieName  = "access_token"
	RefreshCookieName = "refresh_token"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authService := deps.Container.Core.Services.AuthService
		accessToken, accessErr := c.Cookie(AccessCookieName)
		if accessErr == nil {
			claims, err := authService.ValidateAccess(accessToken)
			if err == nil {
				middleWareLogger.Debugf("access granted method=%s path=%s user_id=%s", c.Request.Method, c.Request.URL.Path, claims.Subject)
				setClaims(c, claims)
				c.Next()
				return
			}
			if !errors.Is(err, auth_service.ErrTokenExpired) {
				middleWareLogger.Warnf("access denied: invalid access token method=%s path=%s error=%v", c.Request.Method, c.Request.URL.Path, err)
				abortUnauthorized(c)
				return
			}
			middleWareLogger.Debugf("access token expired, trying refresh method=%s path=%s", c.Request.Method, c.Request.URL.Path)
		} else if !errors.Is(accessErr, http.ErrNoCookie) {
			middleWareLogger.Warnf("access denied: failed to read access cookie method=%s path=%s error=%v", c.Request.Method, c.Request.URL.Path, accessErr)
			abortUnauthorized(c)
			return
		}

		refreshToken, err := c.Cookie(RefreshCookieName)
		if err != nil {
			middleWareLogger.Debugf("access denied: refresh cookie missing method=%s path=%s", c.Request.Method, c.Request.URL.Path)
			abortUnauthorized(c)
			return
		}

		newAccessToken, err := authService.RefreshAccess(c.Request.Context(), refreshToken)
		if err != nil {
			middleWareLogger.Warnf("access denied: token refresh failed method=%s path=%s error=%v", c.Request.Method, c.Request.URL.Path, err)
			abortUnauthorized(c)
			return
		}

		SetAccessCookie(c, newAccessToken)
		setClaims(c, newAccessToken.Claims)
		middleWareLogger.Infof("access token refreshed by middleware method=%s path=%s user_id=%s", c.Request.Method, c.Request.URL.Path, newAccessToken.Claims.Subject)
		c.Next()
	}
}

func ClaimsFromContext(ctx context.Context) (*auth_service.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth_service.Claims)
	return claims, ok
}

func SetAuthCookies(c *gin.Context, pair *auth_service.TokenPair) {
	setCookie(c, AccessCookieName, pair.AccessToken, pair.AccessExpiresAt)
	setCookie(c, RefreshCookieName, pair.RefreshToken, pair.RefreshExpiresAt)
}

func SetAccessCookie(c *gin.Context, token *auth_service.AccessToken) {
	setCookie(c, AccessCookieName, token.Token, token.ExpiresAt)
}

func ClearAuthCookies(c *gin.Context) {
	clearCookie(c, AccessCookieName)
	clearCookie(c, RefreshCookieName)
}

func setClaims(c *gin.Context, claims *auth_service.Claims) {
	ctx := context.WithValue(c.Request.Context(), claimsContextKey, claims)
	c.Request = c.Request.WithContext(ctx)
	c.Set(string(claimsContextKey), claims)
}

func setCookie(c *gin.Context, name, value string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   config.CurrentConfig.Auth.CookieDomain,
		Expires:  expiresAt.UTC(),
		MaxAge:   maxAge(expiresAt),
		HttpOnly: true,
		Secure:   config.CurrentConfig.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Path:     "/",
		Domain:   config.CurrentConfig.Auth.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   config.CurrentConfig.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func maxAge(expiresAt time.Time) int {
	seconds := int(time.Until(expiresAt).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func abortUnauthorized(c *gin.Context) {
	ClearAuthCookies(c)
	c.AbortWithStatusJSON(
		http.StatusUnauthorized,
		common.ErrorResponse(errors.New("authentication required")),
	)
}
