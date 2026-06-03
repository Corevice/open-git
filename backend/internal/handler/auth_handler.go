package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	authUC "github.com/Corevice/open-git/backend/internal/usecase/auth"
)

type AuthHandler struct {
	register *authUC.RegisterUserUsecase
	login    *authUC.LoginUsecase
}

func NewAuthHandler(register *authUC.RegisterUserUsecase, login *authUC.LoginUsecase) *AuthHandler {
	return &AuthHandler{register: register, login: login}
}

type registerRequest struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	user, err := h.register.Execute(c.Request().Context(), authUC.RegisterUserInput{
		Login:    req.Login,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, authUC.ErrDuplicateLogin) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "login already taken"})
		}
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
	}

	return c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid credentials"})
	}

	out, err := h.login.Execute(c.Request().Context(), authUC.LoginInput{
		LoginOrEmail: req.Login,
		Password:     req.Password,
	})
	if err != nil {
		if errors.Is(err, authUC.ErrInvalidCredentials) {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid credentials"})
		}
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid credentials"})
	}

	return c.JSON(http.StatusOK, loginResponse{Token: out.Token})
}
