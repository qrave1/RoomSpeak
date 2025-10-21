package usecase

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/backend/internal/domain/models"
	"github.com/qrave1/RoomSpeak/backend/internal/domain/output"
	"github.com/qrave1/RoomSpeak/backend/internal/infra/adapters/memory"
	repository2 "github.com/qrave1/RoomSpeak/backend/internal/infra/adapters/postgres/repository"
	"golang.org/x/crypto/bcrypt"
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

	// Онлайн пользователи
	GetOnlineUsers(ctx context.Context) ([]output.OnlineUserInfo, error)
}

type userUsecase struct {
	jwtSecret []byte

	userRepo    repository2.UserRepository
	channelRepo repository2.ChannelRepository
	wsRepo      memory.WebsocketConnectionRepository
}

// NewUserUsecase создает новый экземпляр UserUsecase
func NewUserUsecase(
	jwtSecret []byte,
	userRepo repository2.UserRepository,
	channelRepo repository2.ChannelRepository,
	wsRepo memory.WebsocketConnectionRepository,
) UserUsecase {
	return &userUsecase{
		jwtSecret:   jwtSecret,
		userRepo:    userRepo,
		channelRepo: channelRepo,
		wsRepo:      wsRepo,
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

// GetOnlineUsers получает список всех онлайн пользователей
func (uc *userUsecase) GetOnlineUsers(ctx context.Context) ([]output.OnlineUserInfo, error) {
	// Получаем всех подключенных по WebSocket пользователей
	connectedUserIDs := uc.wsRepo.GetAllConnected()

	result := make([]output.OnlineUserInfo, 0, len(connectedUserIDs))

	for _, userID := range connectedUserIDs {
		// Получаем информацию о пользователе
		user, err := uc.userRepo.GetUserByID(userID)
		if err != nil {
			continue // Пропускаем пользователей, которых не можем найти
		}

		info := output.OnlineUserInfo{
			ID:       user.ID.String(),
			Username: user.Username,
		}

		result = append(result, info)
	}

	return result, nil
}
