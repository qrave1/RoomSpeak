package repository

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/qrave1/RoomSpeak/internal/models"
)

type UserRepository interface {
	CreateUser(user models.User) (uuid.UUID, error)
	GetUserByID(id uuid.UUID) (models.User, error)
	GetUserByUsername(username string) (models.User, error)
}

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(user models.User) (uuid.UUID, error) {
	var id uuid.UUID
	query := "INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id"
	err := r.db.QueryRow(query, user.Username, user.Password).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *userRepo) GetUserByID(id uuid.UUID) (models.User, error) {
	var user models.User
	query := "SELECT id, username, password, created_at FROM users WHERE id = $1"
	err := r.db.Get(&user, query, id)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (r *userRepo) GetUserByUsername(username string) (models.User, error) {
	var user models.User
	query := "SELECT id, username, password, created_at FROM users WHERE username = $1"
	err := r.db.Get(&user, query, username)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
