package repository

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	
	"github.com/qrave1/RoomSpeak/internal/domain/models"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByID(id uuid.UUID) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
}

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(user *models.User) error {
	query := "INSERT INTO users (username, password) VALUES ($1, $2)"

	res, err := r.db.Exec(query, user.Username, user.Password)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if aff, err := res.RowsAffected(); aff == 0 || err != nil {
		return fmt.Errorf("create user no rows affected: %w", err)
	}

	return nil
}

func (r *userRepo) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User

	query := "SELECT id, username, password, created_at, updated_at FROM users WHERE id = $1"

	err := r.db.Get(&user, query, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) GetUserByUsername(username string) (*models.User, error) {
	var user models.User

	query := "SELECT id, username, password, created_at, updated_at FROM users WHERE username = $1"

	err := r.db.Get(&user, query, username)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
