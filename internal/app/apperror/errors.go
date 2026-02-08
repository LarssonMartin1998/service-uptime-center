package apperror

import "errors"

var (
	ErrNoServices               = errors.New("config doesn't contain any services, this is not allowed")
	ErrPasswordFileIsEmpty      = errors.New("password file is empty")
	ErrPasswordTooLong          = errors.New("password token too long")
	ErrHeartbeatTimeoutTooShort = errors.New("heartbeat timeout duration too short")
	ErrInvalidServiceName       = errors.New("invalid service name length")
	ErrDuplicateServiceNames    = errors.New("duplicate server names detected, not allowed")
	ErrNoNotifiers              = errors.New("service is missing notifiers, not allowed")
	ErrInvalidNotifProtocol     = errors.New("notification protocol doesn't exist")
)

var (
	CodeInvalidConfig             = 1
	CodeFailedReadingPasswordFile = 2
	CodeInvalidCliArgument        = 3
)
