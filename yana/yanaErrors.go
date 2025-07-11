package yana

type YanaError struct {
	Code int
	Err  error // TODO: Maybe remove and just add a GetErrorStringFromYanaErrorCode() or something??
}

// This is an enum for the error codes of YanaError
// TODO: Add more in the future?
const (
	NoError = iota
	ConnectionFailed
	PingFailed
	QueryFailed
	UserNotFound
	PasswordsNotEqual
	BadClient
	NoteAlreadyExists // Not used yet
)
