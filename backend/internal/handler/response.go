package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type apiResponse struct {
	Data any `json:"data"`
}

type apiErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type apiError struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Details []apiErrorDetail `json:"details"`
}

type apiErrorResponse struct {
	Error     apiError `json:"error"`
	RequestID string   `json:"request_id"`
}

func RespondOK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, apiResponse{Data: data})
}

func RespondError(c echo.Context, statusCode int, code, message, requestID string) error {
	return c.JSON(statusCode, apiErrorResponse{
		Error: apiError{
			Code:    code,
			Message: message,
			Details: []apiErrorDetail{},
		},
		RequestID: requestID,
	})
}
