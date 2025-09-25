package usecase

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/qrave1/RoomSpeak/internal/domain/models"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
)

// UserUsecase определяет интерфейс для работы с пользователями
type UserUsecase interface {
	// Создание пользователя
	CreateUser(ctx context.Context, username, password string) (*models.User, error)

	// Получение пользователей из БД
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// Аутентификация
	ValidateCredentials(ctx context.Context, username, password string) (*models.User, error)
	GenerateJWT(user *models.User) (string, error)
}

type userUsecase struct {
	jwtSecret []byte

	userRepo repository.UserRepository
}

// NewUserUsecase создает новый экземпляр UserUsecase
func NewUserUsecase(jwtSecret []byte, userRepo repository.UserRepository) UserUsecase {
	return &userUsecase{
		jwtSecret: jwtSecret,
		userRepo:  userRepo,
	}
}

// CreateUser создает нового пользователя с хешированным паролем
func (uc *userUsecase) CreateUser(ctx context.Context, username, password string) (*models.User, error) {
	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Создаем пользователя
	user := models.NewUser()
	user.Username = username
	user.Password = string(hashedPassword)

	// Сохраняем в БД
	if err = uc.userRepo.CreateUser(user); err != nil {
		return nil, err
	}

	// Убираем пароль из ответа
	user.Password = ""
	return user, nil
}

// GetUserByID получает пользователя по ID
func (uc *userUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return uc.userRepo.GetUserByID(id)
}

// GetUserByUsername получает пользователя по имени пользователя
func (uc *userUsecase) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return uc.userRepo.GetUserByUsername(username)
}

// ValidateCredentials проверяет учетные данные пользователя
func (uc *userUsecase) ValidateCredentials(ctx context.Context, username, password string) (*models.User, error) {
	// Получаем пользователя из БД
	user, err := uc.userRepo.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Проверяем пароль
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}

	// Убираем пароль из ответа
	user.Password = ""
	return user, nil
}

// GenerateJWT генерирует JWT токен для пользователя
func (uc *userUsecase) GenerateJWT(user *models.User) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(uc.jwtSecret)
}
