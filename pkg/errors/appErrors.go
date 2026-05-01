package errors

import (
	"net/http"
)

type CustomError struct {
	StatusCode   int    `json:"statusCode"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error"`
}

func (e *CustomError) Error() string {
	return e.ErrorMessage
}

func NotFound(err error) error {
	return &CustomError{
		StatusCode:   http.StatusNotFound,
		Message:      "not found",
		ErrorMessage: err.Error(),
	}
}

func BadRequest(err error) error {
	return &CustomError{
		StatusCode:   http.StatusBadRequest,
		Message:      "bad request",
		ErrorMessage: err.Error(),
	}
}

func InternalServerError(err error) error {
	return &CustomError{
		StatusCode:   http.StatusInternalServerError,
		Message:      "internal_server_error",
		ErrorMessage: err.Error(),
	}
}

func Unauthorized(err error) error {
	return &CustomError{
		StatusCode:   http.StatusUnauthorized,
		Message:      "unauthorized",
		ErrorMessage: err.Error(),
	}
}

func Forbidden(err error) error {
	return &CustomError{
		StatusCode:   http.StatusForbidden,
		Message:      "forbidden",
		ErrorMessage: err.Error(),
	}
}

func Conflict(err error) error {
	return &CustomError{
		StatusCode:   http.StatusConflict,
		Message:      "Conflict",
		ErrorMessage: err.Error(),
	}
}

func GatewayTimeout(err error) error {
	return &CustomError{
		StatusCode:   http.StatusGatewayTimeout,
		Message:      "gateway_timeout",
		ErrorMessage: err.Error(),
	}
}
