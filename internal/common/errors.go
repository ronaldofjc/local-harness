package common

import "fmt"

// ApiError representa um erro estruturado retornado pelas tools MCP.
type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ApiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Erros de negocio predefinidos.
var (
	ErrTaskNotFound           = ApiError{Code: "TaskNotFoundError", Message: "task not found"}
	ErrTaskAlreadyCompleted   = ApiError{Code: "TaskAlreadyCompletedError", Message: "task already completed"}
	ErrSpecNotFound           = ApiError{Code: "SpecNotFoundError", Message: "spec not found"}
	ErrSpecParseError         = ApiError{Code: "SpecParseError", Message: "failed to parse spec"}
	ErrSensorNotFound         = ApiError{Code: "SensorNotFoundError", Message: "sensor not found"}
	ErrRubricNotFound         = ApiError{Code: "RubricNotFoundError", Message: "rubric not found"}
	ErrSchemaValidationFailed = ApiError{Code: "SchemaValidationFailedError", Message: "result failed schema validation"}
	ErrSessionNotFound        = ApiError{Code: "SessionNotFoundError", Message: "session not found"}
)
