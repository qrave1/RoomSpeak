package middleware

import (
	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/internal/infra/appctx"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
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
