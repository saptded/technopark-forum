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
