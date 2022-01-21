package usecase

import (
	"github.com/jackc/pgx"
	"technopark-forum/models"
	"technopark-forum/repository"
)

type Service struct {
	repository *repository.Storage
}

// service

func (service *Service) GetStatus() (*models.Status, error) {
	status, err := service.repository.GetStatus()

	return status, err
}
func (service *Service) Clear() error {
	err := service.repository.Clear()
	return err
}

// user

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

// forum

func (service *Service) CreateForum(forum *models.Forum) (*models.Forum, error) {
	user, err := service.repository.GetUserProfile(forum.Author)
	if err != nil {
		return nil, err
	}

	err = service.repository.CreateForum(forum)
	if err != nil {
		if pgError, ok := err.(pgx.PgError); ok && pgError.Code == "23505" {
			newForum, _ := service.repository.GetForum(forum.Slug)
			return newForum, err
		}
		return nil, err
	}
	forum.Author = user.Nickname

	return forum, nil
}

func (service *Service) GetForum(slug string) (*models.Forum, error) {
	forum, err := service.repository.GetForum(slug)

	return forum, err
}

func (service *Service) CreateThread(slug string, threadData *models.Thread) (*models.Thread, error) {
	user, err := service.repository.GetUserProfile(threadData.Author)
	if err != nil {
		return nil, models.UserNotFound(threadData.Author)
	}
	forum, err := service.repository.GetForum(slug)
	if err != nil {
		return nil, models.ForumNotFound(threadData.Slug)
	}
	if threadData.Slug != "" {
		threadExisting, err := service.repository.GetThread(threadData.Slug)
		if err == nil {
			return threadExisting, models.Conflict
		}
	}

	thread, err := service.repository.CreateThread(user, forum, threadData)
	if err != nil {
		if thread != nil {
			return thread, err
		} else {
			return nil, err
		}
		//if pgError, ok := err.(pgx.PgError); ok && pgError.Code == "23505" {
		//	newThread, _ := service.repository.GetThread(strconv.Itoa(threadData.ID))
		//	return newThread, models.Conflict
		//}
		//return nil, err
	}

	thread.Forum = forum.Slug
	thread.Author = user.Nickname
	//thread, err := service.repository.GetThread(strconv.Itoa(threadData.ID))
	//if err != nil {
	//	return nil, err
	//}

	return thread, nil
}

func (service *Service) GetForumUsers(slug string, limit []byte, since []byte, desc []byte) (*models.Users, error) {
	_, err := service.GetForum(slug)
	if err != nil {
		return nil, models.ForumNotFound(slug)
	}

	users, err := service.repository.GetForumUsers(slug, limit, since, desc)
	return users, err
}

func (service *Service) GetForumThreads(slug string, limit []byte, since []byte, desc []byte) (*models.Threads, error) {
	_, err := service.GetForum(slug)
	if err != nil {
		return nil, models.ForumNotFound(slug)
	}

	thread, err := service.repository.GetForumThreads(slug, limit, since, desc)
	return thread, err
}

func (service *Service) CreatePosts(slugOrID interface{}, postsArr *models.Posts) (*models.Posts, error) {
	posts, err := service.repository.CreatePosts(slugOrID, postsArr)

	return posts, err
}

func (service *Service) GetThread(slugOrID interface{}) (*models.Thread, error) {
	thread, err := service.repository.GetThread(slugOrID)

	return thread, err
}

func (service *Service) UpdateThread(slugOrID string, threadUpd *models.ThreadUpdate) (*models.Thread, error) {
	thread, err := service.repository.GetThread(slugOrID)
	if err != nil {
		return nil, err
	}
	thread, err = service.repository.UpdateThread(thread.ID, threadUpd)

	return thread, err
}

func (service *Service) GetThreadPosts(slugOrID *string, limit []byte, since []byte, sort []byte, desc []byte) (*models.Posts, int) {
	posts, status := service.repository.GetThreadPosts(slugOrID, limit, since, sort, desc)

	return posts, status
}

func (service *Service) PutVote(slugOrID interface{}, vote *models.Vote) (*models.Thread, error) {
	thread, err := service.repository.PutVote(slugOrID, vote)

	return thread, err
}

func (service *Service) GetPostDetails(id *string, related []byte) (*models.PostDetails, int) {
	postDetails, status := service.repository.GetPostDetails(id, related)

	return postDetails, status
}

func (service *Service) UpdatePostDetails(id *string, postUpd *models.PostUpdate) (*models.Post, int) {
	post, status := service.repository.UpdatePostDetails(id, postUpd)

	return post, status
}
