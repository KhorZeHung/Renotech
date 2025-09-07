package utils

import (
	"golang.org/x/crypto/bcrypt"
	"renotech.com.my/internal/enum"
)

const (
	MinPasswordLength = 6
	BcryptCost        = 12
)

func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", SystemError(enum.ErrorCodeValidation, "Password must be at least 6 characters long", nil)
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", SystemError(enum.ErrorCodeInternal, "Failed to hash password", nil)
	}

	return string(hashedBytes), nil
}

func VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return SystemError(enum.ErrorCodeValidation, "Invalid password", nil)
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return SystemError(enum.ErrorCodeValidation, "Password must be at least 6 characters long", nil)
	}
	return nil
}
