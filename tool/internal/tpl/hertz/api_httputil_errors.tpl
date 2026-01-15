package httputil

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"{{.Module}}/internal/types"
)

// Common error types
var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrBadRequest    = errors.New("bad request")
	ErrInternal      = errors.New("internal server error")
	ErrRPCFailed     = errors.New("rpc call failed")
)

// ErrorCode represents application-specific error codes
type ErrorCode int

const (
	CodeSuccess            ErrorCode = 0
	CodeBadRequest         ErrorCode = 400
	CodeUnauthorized       ErrorCode = 401
	CodeForbidden          ErrorCode = 403
	CodeNotFound           ErrorCode = 404
	CodeInternalError      ErrorCode = 500
	CodeRPCError           ErrorCode = 502
	CodeServiceUnavailable ErrorCode = 503
)

// AppError represents an application error with code and message
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// HandleError handles errors and sends appropriate HTTP response (exported for use in handler subpackages)
func HandleError(ctx context.Context, c *app.RequestContext, err error) {
	if err == nil {
		return
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		httpStatus := errorCodeToHTTPStatus(appErr.Code)
		c.JSON(httpStatus, types.Response{
			Code:    int(appErr.Code),
			Message: appErr.Message,
		})
		return
	}

	// Handle known error types
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(consts.StatusNotFound, types.Response{
			Code:    http.StatusNotFound,
			Message: "Resource not found",
		})
	case errors.Is(err, ErrUnauthorized):
		c.JSON(consts.StatusUnauthorized, types.Response{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
		})
	case errors.Is(err, ErrForbidden):
		c.JSON(consts.StatusForbidden, types.Response{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		})
	case errors.Is(err, ErrBadRequest):
		c.JSON(consts.StatusBadRequest, types.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
	case errors.Is(err, ErrRPCFailed):
		c.JSON(consts.StatusBadGateway, types.Response{
			Code:    http.StatusBadGateway,
			Message: "Service temporarily unavailable",
		})
	default:
		// Log the actual error for debugging
		// logx.ErrorContext(ctx, "Internal error", zap.Error(err))
		c.JSON(consts.StatusInternalServerError, types.Response{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
		})
	}
}

// errorCodeToHTTPStatus converts ErrorCode to HTTP status code
func errorCodeToHTTPStatus(code ErrorCode) int {
	switch code {
	case CodeSuccess:
		return http.StatusOK
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeRPCError:
		return http.StatusBadGateway
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}