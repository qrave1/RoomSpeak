package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/qrave1/RoomSpeak/backend/internal/infra/appctx"
)

func JWTAuthMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("jwt")
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing or malformed jwt"})
			}

			token, err := jwt.ParseWithClaims(cookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
				return []byte(secret), nil
			})
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired jwt"})
			}

			claims, ok := token.Claims.(*jwt.RegisteredClaims)
			if !ok || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired jwt"})
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid subject"})
			}

			c.SetRequest(
				c.Request().WithContext(
					appctx.WithUserID(c.Request().Context(), userID),
				),
			)

			return next(c)
		}
	}
}

// BuildCookieDomain возвращает значение для cookie.Domain или пустую строку, если Domain не нужно задавать.
//
// host может быть взят из cfg.Domain или r.Host (request.Host).
// prodMode — если true, более строгое поведение (вставляем .example.com для поддоменов).
func BuildCookieDomain(host string, isProd bool) string {
	// Убираем порт если передан: example.com:8080 -> example.com
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}
	host = strings.TrimSpace(host)
	host = strings.ToLower(host)
	if host == "" {
		return ""
	}

	// Если localhost или loopback IP — не указываем Domain
	if host == "localhost" {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil {
		// для реального IP (127.0.0.1, 192.168.x.x) не указываем Domain
		return ""
	}

	// Если это субдомен вида api.example.com -> хотим .example.com
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		// для домена из двух частей (example.com) ставим .example.com
		// для более длинных (sub.api.example.co.uk) логика оставляет последние 2 части:
		// можно усложнить, если надо поддержать co.uk — но это базовый вариант.
		root := strings.Join(parts[len(parts)-2:], ".")
		// Если isProd мы уверены, что хотим корневой домен; в dev можно вернуть host (без dot) или empty.
		if isProd {
			return "." + root
		}
		// dev: если host уже равен root (example.com) — возвращаем .example.com, иначе .root
		return "." + root
	}

	// fallback: ничего
	return ""
}
