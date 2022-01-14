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
