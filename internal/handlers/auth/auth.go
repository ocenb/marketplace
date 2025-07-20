package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/internal/services/auth"
	"github.com/ocenb/marketplace/internal/utils"
	"github.com/ocenb/marketplace/internal/utils/httputil"
)

type AuthHandlerInterface interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	RegisterRoutes(r chi.Router)
}

type RegisterRequest struct {
	Login    string `json:"login" validate:"required,alphanum,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginRequest struct {
	Login    string `json:"login" validate:"required,max=50"`
	Password string `json:"password" validate:"required,max=72"`
}

type LoginResponse struct {
	Token string            `json:"token"`
	User  models.UserPublic `json:"user"`
}

type AuthHandler struct {
	authService auth.AuthServiceInterface
	log         *slog.Logger
	validator   *validator.Validate
}

func New(authService auth.AuthServiceInterface, log *slog.Logger, validator *validator.Validate) AuthHandlerInterface {
	return &AuthHandler{
		authService,
		log,
		validator,
	}
}

// @Summary Register a new user
// @Param user body RegisterRequest true "User registration data"
// @Success 201 {object} models.UserPublic "User registered successfully"
// @Failure 400 {object} httputil.ErrorResponse "Bad request"
// @Failure 409 {object} httputil.ErrorResponse "Conflict"
// @Failure 500 {object} httputil.ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := h.log.With(utils.OpLog("AuthHandler.Register"))

	var req RegisterRequest
	if !httputil.DecodeAndValidate(w, r, &req, h.validator, log) {
		return
	}

	log.Debug("Registration request validated successfully", slog.String("login", req.Login))

	newUser, err := h.authService.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			log.Info("Registration failed", slog.String("login", req.Login), utils.ErrLog(err))
			httputil.ConflictError(w, log, err.Error())
			return
		}
		log.Error("Internal error during registration", utils.ErrLog(err))
		httputil.InternalError(w, log)
		return
	}

	log.Info("User registered successfully", slog.Int64("user_id", newUser.ID), slog.String("login", newUser.Login))

	httputil.WriteJSON(w, newUser,
		http.StatusCreated, log)
}

// @Summary User login
// @Param credentials body LoginRequest true "User login credentials"
// @Success 200 {object} models.UserPublic "Login successful"
// @Failure 400 {object} httputil.ErrorResponse "Bad request"
// @Failure 401 {object} httputil.ErrorResponse "Unauthorized"
// @Failure 500 {object} httputil.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	log := h.log.With(utils.OpLog("AuthHandler.Login"))

	var req LoginRequest
	if !httputil.DecodeAndValidate(w, r, &req, h.validator, log) {
		return
	}

	log.Debug("Login request validated successfully", slog.String("login", req.Login))

	user, token, err := h.authService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrUserNotFound) {
			log.Info("Login failed", slog.String("login", req.Login), utils.ErrLog(err))
			httputil.UnauthorizedError(w, log, auth.ErrInvalidCredentials.Error())
			return
		}
		log.Error("Internal error during user login", utils.ErrLog(err))
		httputil.InternalError(w, log)
		return
	}

	log.Info("User logged in successfully", slog.String("login", req.Login))

	httputil.WriteJSON(w, LoginResponse{
		Token: token,
		User:  *user,
	}, http.StatusOK, log)
}

func (h *AuthHandler) RegisterRoutes(noAuthRouter chi.Router) {
	noAuthRouter.Post("/auth/register", h.Register)
	noAuthRouter.Post("/auth/login", h.Login)
}
