package models

import "github.com/pkg/errors"

//easyjson:json
type ErrorMsg struct {
	Message string `json:"message,omitempty"`
}

var (
	ErrorMessage = func(message error) ErrorMsg { return ErrorMsg{Message: message.Error()} }

	UsersProfileConflict = func(nickname string) error { return errors.Errorf("Conflict on user with nickname %s\n", nickname) }
	UserNotFound         = func(nickname string) error { return errors.Errorf("Can't find user with nickname %s\n", nickname) }
	ForumNotFound        = func(slug string) error { return errors.Errorf("Can't find forum with slug %s\n", slug) }
	Conflict             = errors.New("Entity already exist")
	ThreadNotFound       = errors.New("Thread not found")
	PostNotFound         = errors.New("Post not found")
	UserNotFoundSimple   = errors.New("User not found")
)
