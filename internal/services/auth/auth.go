package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ocenb/marketplace/internal/config"
	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/internal/repos/auth"
	"github.com/ocenb/marketplace/internal/services/user"
	"github.com/ocenb/marketplace/internal/storage"
	"github.com/ocenb/marketplace/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthServiceInterface interface {
	Register(ctx context.Context, login, password string) (*models.UserPublic, error)
	Login(ctx context.Context, login, password string) (*models.UserPublic, string, error)
	ValidateToken(ctx context.Context, token string) (int64, error)
	CleanupExpiredTokens()
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserAlreadyExists  = errors.New("user with this login already exists")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	cfg         *config.Config
	log         *slog.Logger
	authRepo    auth.AuthRepoInterface
	userService user.UserServiceInterface
}

func New(cfg *config.Config, log *slog.Logger, authRepo auth.AuthRepoInterface, userService user.UserServiceInterface) AuthServiceInterface {
	return &AuthService{
		cfg:         cfg,
		log:         log,
		authRepo:    authRepo,
		userService: userService,
	}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (*models.UserPublic, error) {
	var newUser *models.UserPublic

	err := storage.WithTransaction(ctx, s.authRepo, func(txCtx context.Context) error {
		exists, err := s.userService.CheckExists(txCtx, login)
		if err != nil {
			return err
		}
		if exists {
			return ErrUserAlreadyExists
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.JWT.BCryptCost)
		if err != nil {
			return err
		}

		newUser, err = s.userService.Create(txCtx, login, string(hashedPassword))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *AuthService) Login(ctx context.Context, login, password string) (*models.UserPublic, string, error) {
	var token string
	var existingUser *models.User

	err := storage.WithTransaction(ctx, s.authRepo, func(txCtx context.Context) error {
		user, err := s.userService.GetByLogin(txCtx, login)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return err
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			return ErrInvalidCredentials
		}

		token, err = s.createToken(txCtx, user.ID)
		if err != nil {
			return err
		}

		existingUser = user

		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return &models.UserPublic{
		ID:        existingUser.ID,
		Login:     existingUser.Login,
		CreatedAt: existingUser.CreatedAt,
	}, token, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (int64, error) {
	s.log.Debug("Validating token")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			s.log.Error("Invalid token signing method", slog.String("method", token.Method.Alg()))
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWT.JWTSecret), nil
	})
	if err != nil {
		s.log.Error("Token parsing failed", utils.ErrLog(err))
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		s.log.Error("Invalid token", slog.String("token", tokenString))
		return 0, ErrInvalidToken
	}

	userID, ok := claims["userID"].(float64)
	if !ok {
		s.log.Error("Token validation failed: userID not found in token")
		return 0, ErrInvalidToken
	}

	exists, err := s.authRepo.CheckTokenExists(ctx, tokenString)
	if err != nil {
		s.log.Error("Failed to get token", utils.ErrLog(err))
		return 0, err
	}
	if !exists {
		s.log.Info("Token validation failed: token not found or expired", slog.Int64("user_id", int64(userID)))
		return 0, ErrInvalidToken
	}

	s.log.Debug("Token validated successfully", slog.Int64("user_id", int64(userID)))
	return int64(userID), nil
}

func (s *AuthService) CleanupExpiredTokens() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.authRepo.DeleteExpiredTokens(ctx); err != nil {
		s.log.Error("Failed to cleanup expired tokens", utils.ErrLog(err))
	} else {
		s.log.Info("Successfully cleaned up expired tokens")
	}
}

func (s *AuthService) createToken(ctx context.Context, userID int64) (string, error) {
	s.log.Debug("Creating token for user", slog.Int64("user_id", userID))

	expiresAt := time.Now().Add(s.cfg.JWT.TokenLiveTime)

	payload := jwt.MapClaims{
		"userID": userID,
		"exp":    expiresAt.Unix(),
		"iat":    time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.JWTSecret))
	if err != nil {
		s.log.Error("Failed to generate tokens", slog.Int64("user_id", userID), utils.ErrLog(err))
		return "", err
	}

	err = s.authRepo.CreateToken(ctx, tokenString, userID, expiresAt)
	if err != nil {
		s.log.Error("Failed to create token in db", slog.Int64("user_id", userID), utils.ErrLog(err))
		return "", err
	}

	s.log.Info("Token created successfully", slog.Int64("user_id", userID))
	return tokenString, nil
}
