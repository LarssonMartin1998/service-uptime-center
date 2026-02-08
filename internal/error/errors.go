// Package error contains internal error definitions as values and error codes
package error

import "errors"

var (
	ErrNoServices               = errors.New("config doesn't contain any services, this is not allowed")
	ErrPasswordFileIsEmpty      = errors.New("password file is empty")
	ErrPasswordTooLong          = errors.New("password token too long")
	ErrHeartbeatTimeoutTooShort = errors.New("heartbeat timeout duration too short")
	ErrInvalidServiceName       = errors.New("invalid service name length")
)
