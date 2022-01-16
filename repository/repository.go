package repository

import (
	"github.com/jackc/pgx"
	"technopark-forum/models"
)

type Storage struct {
	db *pgx.ConnPool
}

func NewForumStorage(db *pgx.ConnPool) *Storage {
	return &Storage{db: db}
}

func (storage *Storage) CreateUser(user *models.User) (*models.Users, error) {
	queryInsert := `INSERT INTO users (email, nickname, fullname, about) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`

	tx, err := storage.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Rollback()
	}(tx)

	response, err := tx.Exec(queryInsert, user.Email, user.Nickname, user.Fullname, user.About)
	if err != nil {
		return nil, err
	}
	if response.RowsAffected() == 0 {
		querySelect := `SELECT email, nickname, fullname, about FROM users WHERE email=$1 OR nickname=$2`
		rows, err := tx.Query(querySelect, user.Email, user.Nickname)
		if err != nil {
			return nil, err
		}

		users := models.Users{}
		for rows.Next() {
			var currentUser models.User
			err := rows.Scan(&currentUser.Email, &currentUser.Nickname, &currentUser.Fullname, &currentUser.About)
			if err != nil {
				return nil, err
			}
			users = append(users, currentUser)
		}

		rows.Close()
		_ = tx.Rollback()
		return &users, nil
	}

	_ = tx.Commit()
	return nil, nil
}

func (storage *Storage) GetUserProfile(nickname string) (*models.User, error) {
	query := `SELECT email, fullname, about FROM users WHERE nickname = $1`

	user := new(models.User)
	user.Nickname = nickname
	err := storage.db.QueryRow(query, nickname).Scan(&user.Email, &user.Fullname, &user.About)
	if err != nil {
		return nil, models.UserNotFound(nickname)
	}

	return user, nil
}

func (storage *Storage) UpdateUserProfile(oldUser *models.User) (*models.User, error) {
	query := `UPDATE users SET ` +
		`email = COALESCE($1, users.email), fullname = COALESCE($2, users.fullname), about = COALESCE($3, users.about) ` +
		`WHERE nickname=$4 RETURNING email, nickname, fullname, about`

	var (
		newEmail    *string = nil
		newFullname *string = nil
		newAbout    *string = nil
	)
	if oldUser.Email != "" {
		newEmail = &oldUser.Email
	}
	if oldUser.Fullname != "" {
		newFullname = &oldUser.Fullname
	}
	if oldUser.About != "" {
		newAbout = &oldUser.About
	}

	newUser := new(models.User)
	err := storage.db.QueryRow(query, &newEmail, &newFullname, &newAbout, oldUser.Nickname).
		Scan(&newUser.Email, &newUser.Nickname, &newUser.Fullname, &newUser.About)
	if err != nil {
		if _, ok := err.(pgx.PgError); ok {
			return nil, models.UsersProfileConflict(oldUser.Nickname)
		}
		return nil, models.UserNotFound(oldUser.Nickname)
	}

	return newUser, nil
}
