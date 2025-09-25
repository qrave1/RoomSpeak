package dto

import "github.com/google/uuid"

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type GetMeResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}
