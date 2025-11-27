package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

const (
	CodeInternalError   = 1000
	CodeValidationError = 1001
	CodeNotFound        = 1002
	CodeDuplicateEntry  = 1003
)

func WriteError(w http.ResponseWriter, statuCode int, errorCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statuCode)

	errorMsg := ""
	switch statuCode {
	case http.StatusBadRequest:
		errorMsg = "Bad Request"
	case http.StatusNotFound:
		errorMsg = "Not Found"
	case http.StatusInternalServerError:
		errorMsg = "Internal Server Error"
	case http.StatusConflict:
		errorMsg = "Conflict"
	default:
		errorMsg = "Error"
	}

	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorMsg,
		Message: message,
		Code:    errorCode,
	})
}

func DatabaseError(w http.ResponseWriter, err error, operation string) {
	errorMsg := err.Error()
	if strings.Contains(errorMsg, "duplicate key value violates unique constraint") || strings.Contains(errorMsg, "повторяющееся значение ключа нарушает ограничение уникальности") {
		WriteError(w, http.StatusConflict, CodeDuplicateEntry, "user with this email already exist")
		return
	}

	fmt.Printf("Database error in %s: %v\n", operation, err)

	WriteError(w, http.StatusInternalServerError, CodeInternalError, "database operation failed")
}
