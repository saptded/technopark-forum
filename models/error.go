package models

import "github.com/pkg/errors"

//easyjson:json
type ErrorMsg struct {
	Message string `json:"message,omitempty"`
}

var (
	ErrorMessage = func(message error) ErrorMsg { return ErrorMsg{Message: message.Error()} }

	UsersProfileConflict = func(nickname string) error { return errors.Errorf("Can't find user with nickname %s\n", nickname) }
	UserNotFound         = func(nickname string) error { return errors.Errorf("Can't find user with nickname %s\n", nickname) }
)
