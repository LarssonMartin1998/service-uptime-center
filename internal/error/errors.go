// Package error contains internal error definitions as values and error codes
package error

import "errors"

var (
	ErrNoServiceGroups    = errors.New("config doesn't contain any service groups, this is not allowed")
	ErrPasswordTooLong    = errors.New("password token too long")
	ErrHeartbeatTooShort  = errors.New("heartbeat frequency too short")
	ErrNoServices         = errors.New("service group has no services")
	ErrInvalidServiceName = errors.New("invalid service name length")
)
