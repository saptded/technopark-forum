package usecase

import (
	"technopark-forum/models"
	"technopark-forum/repository"
)

type Service struct {
	repository *repository.Storage
}

func NewForumService(repository *repository.Storage) *Service {
	return &Service{repository: repository}
}

func (service *Service) CreateUser(user *models.User) (*models.Users, error) {
	users, err := service.repository.CreateUser(user)

	return users, err
}

func (service *Service) GetUserProfile(nickname string) (*models.User, error) {
	user, err := service.repository.GetUserProfile(nickname)

	return user, err
}

func (service *Service) UpdateUserProfile(oldUser *models.User) (*models.User, error) {
	newUser, err := service.repository.UpdateUserProfile(oldUser)

	return newUser, err
}
