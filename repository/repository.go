package repository

import (
	"bytes"
	"context"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/jackc/pgx"
	"log"
	"net/http"
	"strconv"
	"strings"
	"technopark-forum/models"
	"time"
)

type Storage struct {
	db *pgx.ConnPool
}

func NewForumStorage(db *pgx.ConnPool) *Storage {
	return &Storage{db: db}
}

// service

func (storage *Storage) GetStatus() (*models.Status, error) {
	tx, err := storage.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Commit()
	}(tx)

	status := new(models.Status)
	err = tx.QueryRow("SELECT (SELECT count(*) FROM forums) as forums, (SELECT count(*) FROM posts) as posts, (SELECT count(*) FROM users) as users, (SELECT count(*) FROM threads) as threads").Scan(&status.Forum, &status.Post, &status.User, &status.Thread)
	if err != nil {
		return nil, err
	}

	return status, nil
}

func (storage *Storage) Clear() error {
	tx, err := storage.db.Begin()
	if err != nil {
		return err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Commit()
	}(tx)

	//_, err = tx.Exec("TRUNCATE forum_users, vote, post, thread, forum, users RESTART IDENTITY CASCADE")
	_, err = tx.Exec("TRUNCATE forum_users, posts, threads, forums, users RESTART IDENTITY CASCADE")
	if err != nil {
		return err
	}

	return nil
}

// user

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
	query := `SELECT nickname, email, fullname, about FROM users WHERE nickname = $1`

	user := new(models.User)
	user.Nickname = nickname
	err := storage.db.QueryRow(query, nickname).Scan(&user.Nickname, &user.Email, &user.Fullname, &user.About)
	if err != nil {
		return nil, models.UserNotFound(nickname)
	}

	return user, nil
}

func (storage *Storage) UpdateUserProfile(oldUser *models.User) (*models.User, error) {
	query := `UPDATE users SET ` +
		`email = COALESCE($1, users.email), fullname = COALESCE($2, users.fullname), about = COALESCE($3, users.about) ` +
		`WHERE nickname=$4 RETURNING email::TEXT, nickname::TEXT, fullname, about`

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

// forum

func (storage *Storage) CreateForum(forum *models.Forum) error {
	query := `INSERT INTO forums(title, author, slug) values ($1, (SELECT nickname FROM users WHERE nickname=$2), $3);`

	tx, err := storage.db.Begin()
	if err != nil {
		return err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Rollback()
	}(tx)

	_, err = tx.Exec(query, forum.Title, forum.Author, forum.Slug)
	if err != nil {
		return err
	}

	_ = tx.Commit()
	return nil
}

func (storage *Storage) GetForum(slug string) (*models.Forum, error) {
	query := `SELECT title, slug::TEXT, author::TEXT, posts, threads FROM forums WHERE slug=$1`

	forum := new(models.Forum)

	err := storage.db.QueryRow(query, slug).
		Scan(&forum.Title, &forum.Slug, &forum.Author, &forum.Posts, &forum.Threads)
	if err != nil {
		return nil, err
	}

	return forum, err
}

func (storage *Storage) CreateThread(user *models.User, forum *models.Forum, thread *models.Thread) (*models.Thread, error) {
	query := `INSERT INTO threads(title, author, forum, message, slug, created_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING RETURNING id`

	tx, err := storage.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Rollback()
	}(tx)

	err = tx.QueryRow(query, thread.Title, thread.Author, forum.Slug,
		thread.Message, thread.Slug, thread.Created).
		Scan(&thread.ID)
	if err != nil {
		existingThread := new(models.Thread)
		queryExists := `SELECT id, slug::TEXT, title, message, forum::TEXT, author::TEXT, created_at, votes FROM threads WHERE slug=$1`

		if err = tx.QueryRow(queryExists, thread.Slug).
			Scan(&existingThread.ID, &existingThread.Slug, &existingThread.Title,
				&existingThread.Message, &existingThread.Forum, &existingThread.Author, &existingThread.Created,
				&existingThread.Votes); err == nil {

			_ = tx.Rollback()
			return existingThread, models.Conflict
		}

		return nil, err
	}

	queryUpdateForumUsers := `INSERT INTO forum_users(nickname, forum) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, _ = storage.db.Exec(queryUpdateForumUsers, thread.Author, thread.Forum)
	queryUpdateForum := `UPDATE forums SET threads = threads + 1 WHERE slug =$1`
	_, _ = storage.db.Exec(queryUpdateForum, thread.Forum)

	_ = tx.Commit()
	return thread, nil
}

func (storage *Storage) GetThread(slugOrID interface{}) (*models.Thread, error) {
	queryBySlug := `SELECT id, title, author, forum, message, votes, slug, created_at FROM threads WHERE slug=$1`
	queryByID := `SELECT id, title, author, forum, message, votes, slug, created_at FROM threads WHERE id=$1`

	thread := new(models.Thread)

	_, err := strconv.Atoi(slugOrID.(string))

	if err != nil {
		err = storage.db.QueryRow(queryBySlug, slugOrID).
			Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &thread.Votes, &thread.Slug, &thread.Created)
		if err != nil {
			return nil, err
		}
	} else {
		err = storage.db.QueryRow(queryByID, slugOrID).
			Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &thread.Votes, &thread.Slug, &thread.Created)
		if err != nil {
			return nil, err
		}
	}

	return thread, nil
}

func (storage *Storage) GetForumUsers(slug interface{}, limit []byte, since []byte, desc []byte) (*models.Users, error) {
	queryDesc := `SELECT email, forum_users.nickname, fullname, about FROM forum_users 
JOIN users u on forum_users.nickname = u.nickname WHERE forum_users.forum=$1 ORDER BY lower(forum_users.nickname) DESC LIMIT $2::TEXT::INTEGER`
	querySinceDesc := `SELECT email, forum_users.nickname, fullname, about FROM forum_users 
JOIN users u on forum_users.nickname = u.nickname WHERE forum_users.forum=$1 AND lower(forum_users.forum) > lower($2) ORDER BY lower(forum_users.nickname) DESC LIMIT $3::TEXT::INTEGER`
	querySince := `SELECT email, forum_users.nickname, fullname, about FROM forum_users 
JOIN users u on forum_users.nickname = u.nickname WHERE forum_users.forum=$1 AND lower(forum_users.forum) > lower($2) ORDER BY lower(forum_users.nickname) LIMIT $3::TEXT::INTEGER`
	query := `SELECT email, forum_users.nickname, fullname, about FROM forum_users 
JOIN users u on forum_users.nickname = u.nickname WHERE forum_users.forum=$1 ORDER BY lower(forum_users.nickname) LIMIT $2::TEXT::INTEGER`

	var err error
	var rows *pgx.Rows
	if since == nil {
		if bytes.Equal([]byte("true"), desc) {
			rows, err = storage.db.Query(queryDesc, slug, limit)
		} else {
			rows, err = storage.db.Query(query, slug, limit)
		}
	} else {
		if bytes.Equal([]byte("true"), desc) {
			rows, err = storage.db.Query(querySinceDesc, slug, since, limit)
		} else {
			rows, err = storage.db.Query(querySince, slug, since, limit)
		}
	}
	if err != nil {
		rows.Close()
		return nil, err
	}
	var users models.Users

	for rows.Next() {
		user := new(models.User)
		if err = rows.Scan(&user.Nickname, &user.Email, &user.About, &user.Fullname); err != nil {
			rows.Close()
			return nil, err
		}
		users = append(users, *user)
	}

	rows.Close()

	return &users, nil
}

func (storage *Storage) GetForumThreads(slug interface{}, limit []byte, since []byte, desc []byte) (*models.Threads, error) {
	queryDesc := `SELECT id, slug, title, message, forum, author, created_at, votes FROM threads
WHERE forum = $1 ORDER BY created_at DESC LIMIT $2::TEXT::INTEGER`
	querySinceDesc := `SELECT id, slug, title, message, forum, author, created_at, votes FROM threads
WHERE forum = $1 AND created_at <= $2::TEXT::TIMESTAMPTZ ORDER BY created_at DESC LIMIT $3::TEXT::INTEGER`
	querySince := `SELECT id, slug, title, message, forum, author, created_at, votes FROM threads
WHERE forum = $1 AND created_at >= $2::TEXT::TIMESTAMPTZ ORDER BY created_at LIMIT $3::TEXT::INTEGER`
	query := `SELECT id, slug, title, message, forum, author, created_at, votes FROM threads
WHERE forum = $1 ORDER BY created_at LIMIT $2::TEXT::INTEGER`

	var err error
	var rows *pgx.Rows

	if since == nil {
		if bytes.Equal([]byte("true"), desc) {
			rows, err = storage.db.Query(queryDesc, slug, limit)
		} else {
			rows, err = storage.db.Query(query, slug, limit)
		}
	} else {
		if bytes.Equal([]byte("true"), desc) {
			rows, err = storage.db.Query(querySinceDesc, slug, since, limit)
		} else {
			rows, err = storage.db.Query(querySince, slug, since, limit)
		}
	}

	if err != nil {
		rows.Close()
		return nil, err
	}

	var threads models.Threads
	for rows.Next() {
		thread := new(models.Thread)
		if err = rows.Scan(&thread.ID, &thread.Slug, &thread.Title, &thread.Message,
			&thread.Forum, &thread.Author, &thread.Created, &thread.Votes); err != nil {
			return nil, err
		}

		threads = append(threads, *thread)
	}
	rows.Close()

	return &threads, nil
}

// threads

func (storage *Storage) CreatePosts(slugOrID interface{}, posts *models.Posts) (*models.Posts, error) {
	if len(*posts) == 0 {
		return nil, nil
	}

	queryBySlug := `SELECT id, forum FROM threads WHERE slug=$1`
	queryByID := `SELECT id, forum FROM threads WHERE id=$1`

	tx, err := storage.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Rollback()
	}(tx)

	batch := tx.BeginBatch()
	created := time.Unix(0, 0)

	var (
		forumSlug string
	)

	threadIdentifier, err := strconv.Atoi(slugOrID.(string))
	if err != nil {
		if err = tx.QueryRow(queryBySlug, slugOrID).Scan(&threadIdentifier, &forumSlug); err != nil {
			return nil, models.ThreadNotFound
		}
	} else {
		if err = tx.QueryRow(queryByID, threadIdentifier).Scan(&threadIdentifier, &forumSlug); err != nil {
			return nil, models.ThreadNotFound
		}
	}

	//queryGetForum := `SELECT id FROM forums WHERE slug=$1`
	//if err = tx.QueryRow(queryGetForum, &forumSlug).Scan(&forumID); err != nil {
	//	return nil, err
	//}

	query := `SELECT array_agg(nextval('posts_id_seq')::BIGINT) FROM generate_series(1, $1)`
	ids := make([]int64, 0, len(*posts))
	if err = tx.QueryRow(query, len(*posts)).Scan(&ids); err != nil {
		return nil, err
	}

	comparator := func(lhs, rhs interface{}) int {
		return strings.Compare(lhs.(string), rhs.(string))
	}

	var postsNeedParents []int
	authorSet := treeset.NewWith(comparator)

	querySelectParents := `SELECT thread, parents FROM posts WHERE id = $1`
	if _, err := tx.Prepare("selectParentAndParents", querySelectParents); err != nil {
		return nil, err
	}
	for i, post := range *posts {
		authorSet.Add(strings.ToLower(post.Author))
		if post.Parent != 0 {
			postsNeedParents = append(postsNeedParents, i)
			batch.Queue("selectParentAndParents", []interface{}{int(post.Parent)}, nil, nil)
		}
	}

	queryGetProfiles := `SELECT nickname::TEXT, email::TEXT, about, fullname FROM users WHERE nickname = $1`
	if _, err := tx.Prepare("getUserProfileQuery", queryGetProfiles); err != nil {
		return nil, err
	}

	authorOrderedSet := authorSet.Values()
	for _, nickname := range authorOrderedSet {
		batch.Queue("getUserProfileQuery", []interface{}{nickname}, nil, nil)
	}

	var parentThreadID int64
	if err = batch.Send(context.Background(), nil); err != nil {
		return nil, err
	}

	for _, postIdx := range postsNeedParents {
		if err = batch.QueryRowResults().
			Scan(&parentThreadID, &(*posts)[postIdx].Parents); err != nil {
			return nil, models.Conflict
		}
		if parentThreadID != 0 && parentThreadID != int64(threadIdentifier) {
			return nil, models.Conflict
		}
	}

	authorRealNicknameMap := make(map[string]string)
	var userModelsOrderedSet models.Users

	for _, userNickname := range authorOrderedSet {
		user := models.User{}
		if err = batch.QueryRowResults().
			Scan(&user.Nickname, &user.Email, &user.About, &user.Fullname); err != nil {
			return nil, models.UserNotFoundSimple
		}
		userModelsOrderedSet = append(userModelsOrderedSet, user)
		authorRealNicknameMap[userNickname.(string)] = user.Nickname
	}

	queryInsertPost := `INSERT INTO posts(id, author, message, created_at, forum, thread, parent, parents, main_parent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING created_at`
	if _, err := tx.Prepare("insertIntoPost", queryInsertPost); err != nil {
		return nil, err
	}
	currentPosts := new(models.Posts)
	for index, post := range *posts {
		post.ID = int(ids[index])
		post.Thread = threadIdentifier
		post.Forum = forumSlug
		post.Created = created
		post.Author = authorRealNicknameMap[strings.ToLower(post.Author)]
		post.Parents = append(post.Parents, int32(ids[index]))
		batch.Queue("insertIntoPost", []interface{}{post.ID, post.Author, post.Message, post.Created, post.Forum, post.Thread, post.Parent, post.Parents, post.Parents[0]}, nil, nil)
		*currentPosts = append(*currentPosts, post)
	}

	queryInsertForumUser := `INSERT INTO forum_users(forum, nickname) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	if _, err := tx.Prepare("insertIntoForumUsers", queryInsertForumUser); err != nil {
		return nil, err
	}
	for _, user := range userModelsOrderedSet {
		batch.Queue("insertIntoForumUsers", []interface{}{forumSlug, user.Nickname}, nil, nil)
	}
	if err = batch.Send(context.Background(), nil); err != nil {
		return nil, err
	}

	for range *posts {
		if _, err := batch.ExecResults(); err != nil {
			return nil, err
		}
	}

	for range userModelsOrderedSet {
		if _, err := batch.ExecResults(); err != nil {
			return nil, err
		}
	}

	queryUpdate := `UPDATE forums SET posts=posts+$2 WHERE slug=$1`
	_, err = tx.Exec(queryUpdate, forumSlug, len(*posts))
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	_ = tx.Commit()
	return currentPosts, nil
}

func (storage *Storage) UpdateThread(threadID int, threadUpdate *models.ThreadUpdate) (*models.Thread, error) {
	query := `UPDATE threads SET message = coalesce($1, message), title = coalesce($2,title) WHERE id = $3 
RETURNING  id, slug, title, message, forum, author, created_at, votes`

	tx, err := storage.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Commit()
	}(tx)

	//var ID int
	//var fs string
	//if ID, err = strconv.Atoi(*slugOrID); err != nil {
	//	if err = tx.QueryRow("SELECT id, forum::TEXT FROM thread WHERE slug=$1", slugOrID).Scan(&ID, &fs);
	//		err != nil {
	//		return nil, http.StatusNotFound
	//	}
	//}

	thread := new(models.Thread)

	if err = tx.QueryRow(query, threadUpdate.Message, threadUpdate.Title, threadID).
		Scan(&thread.ID, &thread.Slug, &thread.Title, &thread.Message, &thread.Forum,
			&thread.Author, &thread.Created, &thread.Votes); err != nil {
		return nil, err
	}
	return thread, nil
}

func (storage *Storage) GetThreadPosts(slugOrID *string, limit []byte, since []byte, sort []byte, desc []byte) (*models.Posts, int) {
	queryByID := `SELECT id FROM threads WHERE id=$1`
	queryBySlug := `SELECT id FROM threads WHERE slug=$1`

	var ID int
	if _, err := strconv.Atoi(*slugOrID); err != nil {
		if err = storage.db.QueryRow(queryBySlug, slugOrID).Scan(&ID); err != nil {
			return nil, http.StatusNotFound
		}
	} else {
		if err = storage.db.QueryRow(queryByID, slugOrID).Scan(&ID); err != nil {
			return nil, http.StatusNotFound
		}
	}

	switch true {
	case bytes.Equal([]byte("tree"), sort):
		postsTree, status := getThreadPostsTree(storage, ID, limit, since, desc)
		return postsTree, status
	case bytes.Equal([]byte("parent_tree"), sort):
		postsParentTree, status := getThreadPostsParentTree(storage, ID, limit, since, desc)
		return postsParentTree, status
	default:
		PostsFlat, status := getThreadPostsFlat(storage, ID, limit, since, desc)
		return PostsFlat, status
	}
}

func getThreadPostsTree(storage *Storage, ID int, limit []byte, since []byte, desc []byte) (*models.Posts, int) {
	getPostsTreeSinceLimitDesc := `SELECT id, author::TEXT, message, created_at, forum::TEXT, thread, is_edited, parent FROM posts
WHERE thread = $1 AND parents < (SELECT parents FROM posts WHERE id = $3::TEXT::INTEGER) ORDER BY parents DESC LIMIT $2::TEXT::BIGINT`
	getPostsTreeSinceLimit := `SELECT id, author::TEXT, message, created_at, forum::TEXT, thread, is_edited, parent
FROM posts WHERE thread = $1 AND parents > (SELECT parents FROM posts WHERE id = $3::TEXT::INTEGER) ORDER BY parents LIMIT $2::TEXT::BIGINT`
	getPostsTreeLimitDesc := `SELECT id, author::TEXT, message, created_at, forum::TEXT, thread, is_edited, parent FROM posts
WHERE thread = $1 ORDER BY parents DESC LIMIT $2::TEXT::BIGINT`
	getPostsTreeLimit := `SELECT id, author::TEXT, message, created_at, forum::TEXT, thread, is_edited, parent FROM posts
WHERE thread = $1 ORDER BY parents LIMIT $2::TEXT::BIGINT`

	var (
		err  error
		rows *pgx.Rows
	)
	if since != nil {
		if limit != nil {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsTreeSinceLimitDesc, ID, limit, since)
			} else {
				rows, err = storage.db.Query(getPostsTreeSinceLimit, ID, limit, since)
			}
		} else {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsTreeSinceLimitDesc, ID, nil, since)
			} else {
				rows, err = storage.db.Query(getPostsTreeSinceLimit, ID, nil, since)
			}
		}
	} else {
		if limit != nil {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsTreeLimitDesc, ID, limit)
			} else {
				rows, err = storage.db.Query(getPostsTreeLimit, ID, limit)
			}
		} else {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsTreeLimitDesc, ID, nil)
			} else {
				rows, err = storage.db.Query(getPostsTreeLimit, ID, nil)
			}
		}
	}

	if err != nil {
		return nil, http.StatusInternalServerError
	}

	var posts models.Posts

	for rows.Next() {
		post := new(models.Post)

		if err = rows.Scan(&post.ID, &post.Author, &post.Message,
			&post.Created, &post.Forum, &post.Thread,
			&post.IsEdited, &post.Parent); err != nil {
			return nil, http.StatusInternalServerError
		}
		posts = append(posts, *post)
	}
	rows.Close()

	return &posts, http.StatusOK
}

func getThreadPostsParentTree(storage *Storage, ID int, limit []byte, since []byte, desc []byte) (*models.Posts, int) {
	getPostsParentTreeSinceLimitDesc := `SELECT p.id,
	p.author::TEXT,
	p.message,
	p.created_at,
	p.forum::TEXT,
	p.thread,
	p.is_edited,
	p.parent
FROM posts p
JOIN (
	SELECT id
	FROM posts
	WHERE parent=0
		AND thread = $1
		AND main_parent < (SELECT main_parent
			FROM posts
			WHERE id = $3::TEXT::INTEGER)
	ORDER BY id DESC
	LIMIT $2::TEXT::INTEGER) s
ON p.main_parent=s.id
ORDER BY p.parents[1] DESC, p.parents[2:]`
	getPostsParentTreeSinceLimit := `SELECT p.id,
	p.author::TEXT,
	p.message,
	p.created_at,
	p.forum::TEXT,
	p.thread,
	p.is_edited,
	p.parent
FROM posts p
JOIN (
	SELECT id
	FROM posts
	WHERE parent=0
		AND thread = $1
		AND main_parent > (SELECT main_parent
			FROM posts
			WHERE id = $3::TEXT::INTEGER)
	ORDER BY id
	LIMIT $2::TEXT::INTEGER) s
ON p.main_parent=s.id
ORDER BY p.parents`
	getPostsParentTreeLimitDesc := `SELECT p.id,
	p.author::TEXT,
	p.message,
	p.created_at,
	p.forum::TEXT,
	p.thread,
	p.is_edited,
	p.parent
FROM posts p
JOIN (
	SELECT id
	FROM posts
	WHERE parent=0 AND thread = $1
	ORDER BY id DESC
	LIMIT $2::TEXT::INTEGER) s
ON p.main_parent=s.id
ORDER BY p.parents[1] DESC, p.parents[2:]`
	getPostsParentTreeLimit := `SELECT p.id,
	p.author::TEXT,
	p.message,
	p.created_at,
	p.forum::TEXT,
	p.thread,
	p.is_edited,
	p.parent
FROM posts p
JOIN (
	SELECT id
	FROM posts
	WHERE parent=0 AND thread = $1
	ORDER BY id
	LIMIT $2::TEXT::INTEGER) s
ON p.main_parent=s.id
ORDER BY p.parents`
	var (
		err  error
		rows *pgx.Rows
	)

	if since != nil {
		if limit != nil {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsParentTreeSinceLimitDesc, ID, limit, since)
			} else {
				rows, err = storage.db.Query(getPostsParentTreeSinceLimit, ID, limit, since)
			}
		} else {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsParentTreeSinceLimitDesc, ID, nil, since)
			} else {
				rows, err = storage.db.Query(getPostsParentTreeSinceLimit, ID, nil, since)
			}
		}
	} else {
		if limit != nil {
			if bytes.Equal(desc, []byte("true")) {
				/*men*/ rows, err = storage.db.Query(getPostsParentTreeLimitDesc, ID, limit)
			} else {
				rows, err = storage.db.Query(getPostsParentTreeLimit, ID, limit)
			}
		} else {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsParentTreeLimitDesc, ID, nil)
			} else {
				rows, err = storage.db.Query(getPostsParentTreeLimit, ID, nil)
			}
		}
	}

	if err != nil {
		return nil, http.StatusInternalServerError
	}

	var posts models.Posts

	for rows.Next() {
		post := new(models.Post)

		if err = rows.Scan(&post.ID, &post.Author, &post.Message,
			&post.Created, &post.Forum, &post.Thread,
			&post.IsEdited, &post.Parent); err != nil {
			return nil, http.StatusInternalServerError
		}
		posts = append(posts, *post)
	}
	rows.Close()

	return &posts, http.StatusOK
}

func getThreadPostsFlat(storage *Storage, ID int, limit []byte, since []byte, desc []byte) (*models.Posts, int) {
	getPostsFlatSinceLimitDesc := `SELECT id,
	author::TEXT,
	message,
	created_at,
	forum::TEXT,
	thread,
	is_edited,
	parent
FROM posts
WHERE thread=$1
	AND id < $3::TEXT::INTEGER
ORDER BY id DESC
LIMIT $2::TEXT::BIGINT`
	getPostsFlatSinceLimit := `SELECT id,
	author::TEXT,
	message,
	created_at,
	forum::TEXT,
	thread,
	is_edited,
	parent
FROM posts
WHERE thread=$1
	AND id > $3::TEXT::INTEGER
ORDER BY id
LIMIT $2::TEXT::BIGINT`
	getPostsFlatLimitDesc := `SELECT id,
	author::TEXT,
	message,
	created_at,
	forum::TEXT,
	thread,
	is_edited,
	parent
FROM posts
WHERE thread=$1
ORDER BY id DESC
LIMIT $2::TEXT::BIGINT`
	getPostsFlatLimit := `SELECT id,
	author::TEXT,
	message,
	created_at,
	forum::TEXT,
	thread,
	is_edited,
	parent
FROM posts
WHERE thread=$1
ORDER BY id
LIMIT $2::TEXT::BIGINT`

	var (
		err  error
		rows *pgx.Rows
	)

	if since != nil {
		if limit != nil {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsFlatSinceLimitDesc, ID, limit, since)
			} else {
				rows, err = storage.db.Query(getPostsFlatSinceLimit, ID, limit, since)
			}
		} else {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsFlatSinceLimitDesc, ID, nil, since)
			} else {
				rows, err = storage.db.Query(getPostsFlatSinceLimit, ID, nil, since)
			}
		}
	} else {
		if limit != nil {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsFlatLimitDesc, ID, limit)
			} else {
				rows, err = storage.db.Query(getPostsFlatLimit, ID, limit)
			}
		} else {
			if bytes.Equal(desc, []byte("true")) {
				rows, err = storage.db.Query(getPostsFlatLimitDesc, ID, nil)
			} else {
				rows, err = storage.db.Query(getPostsFlatLimit, ID, nil)
			}
		}
	}

	if err != nil {
		return nil, http.StatusInternalServerError
	}

	var posts models.Posts

	for rows.Next() {
		post := new(models.Post)

		if err = rows.Scan(&post.ID, &post.Author, &post.Message,
			&post.Created, &post.Forum, &post.Thread,
			&post.IsEdited, &post.Parent); err != nil {
			return nil, http.StatusInternalServerError
		}
		posts = append(posts, *post)
	}
	rows.Close()

	return &posts, http.StatusOK
}

func (storage *Storage) PutVote(slugOrID interface{}, vote *models.Vote) (*models.Thread, error) {
	putVoteByThreadSlug := `WITH sub AS (
	INSERT INTO votes (user_nickname, thread_id, voice)
	VALUES (
		$1,
		(SELECT id FROM threads WHERE slug=$2),
		$3)
	ON CONFLICT ON CONSTRAINT unique_user_and_thread
	DO UPDATE
		SET prev_voice = votes.voice ,
			voice = EXCLUDED.voice
	RETURNING prev_voice,
		voice,
		thread_id)
UPDATE threads
SET votes = votes - (SELECT prev_voice-voice FROM sub)
WHERE slug=$2
RETURNING id,
	slug::TEXT,
	title,
	message,
	forum::TEXT,
	author::TEXT,
	created_at,
	votes`
	putVoteByThreadID := `WITH sub AS (
	INSERT INTO votes (user_nickname, thread_id, voice)
	VALUES (
		$1,
		$2, $3)
	ON CONFLICT ON CONSTRAINT unique_user_and_thread
		DO UPDATE
			SET prev_voice = votes.voice ,
				voice = EXCLUDED.voice
	RETURNING prev_voice, voice, thread_id)
UPDATE threads
SET votes = votes - (SELECT prev_voice-voice FROM sub)
WHERE id = $2
RETURNING id,
	slug::TEXT,
	title, message,
	forum::TEXT,
	author::TEXT,
	created_at,
	votes`

	tx, err := storage.db.Begin()
	if err != nil {
		log.Fatalln(err)
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Commit()
	}(tx)

	_, err = strconv.Atoi(slugOrID.(string))

	thread := new(models.Thread)

	if err != nil {
		err = tx.QueryRow(putVoteByThreadSlug, vote.Nickname, slugOrID, vote.Voice).Scan(&thread.ID, &thread.Slug, &thread.Title, &thread.Message, &thread.Forum, &thread.Author, &thread.Created, &thread.Votes)
	} else {
		err = tx.QueryRow(putVoteByThreadID, vote.Nickname, slugOrID, vote.Voice).Scan(&thread.ID, &thread.Slug, &thread.Title, &thread.Message, &thread.Forum, &thread.Author, &thread.Created, &thread.Votes)
	}

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return thread, nil
}

// post

func (storage *Storage) GetPostDetails(id *string, related []byte) (*models.PostDetails, int) {
	queryUsers := `SELECT nickname::TEXT, email::TEXT, about, fullname FROM users WHERE nickname = $1`
	queryForum := `SELECT slug::TEXT, title, posts, threads, author::TEXT FROM forums WHERE slug=$1`
	queryThread := `SELECT id, slug::TEXT, title, message, forum::TEXT, author::TEXT, created_at, votes FROM threads WHERE id=$1`
	queryPost := `SELECT id, author::TEXT, message, created_at, forum::TEXT, thread, is_edited, parent FROM posts WHERE id=$1`

	postDetails := models.PostDetails{}
	postDetails.PostDetails = &models.Post{}

	err := storage.db.QueryRow(queryPost, id).
		Scan(&postDetails.PostDetails.ID, &postDetails.PostDetails.Author,
			&postDetails.PostDetails.Message, &postDetails.PostDetails.Created,
			&postDetails.PostDetails.Forum, &postDetails.PostDetails.Thread,
			&postDetails.PostDetails.IsEdited, &postDetails.PostDetails.Parent)
	if err != nil {
		return nil, http.StatusNotFound
	}

	if related == nil {
		return &postDetails, http.StatusOK
	}

	relatedArr := strings.Split(string(related), ",")

	for _, val := range relatedArr {
		switch val {
		case "user":
			postDetails.AuthorDetails = &models.User{}
			storage.db.QueryRow(queryUsers, &postDetails.PostDetails.Author).
				Scan(&postDetails.AuthorDetails.Nickname, &postDetails.AuthorDetails.Email,
					&postDetails.AuthorDetails.About, &postDetails.AuthorDetails.Fullname)
		case "forum":
			postDetails.ForumDetails = &models.Forum{}
			storage.db.QueryRow(queryForum, postDetails.PostDetails.Forum).
				Scan(&postDetails.ForumDetails.Slug, &postDetails.ForumDetails.Title,
					&postDetails.ForumDetails.Posts, &postDetails.ForumDetails.Threads,
					&postDetails.ForumDetails.Author)
		case "thread":
			postDetails.ThreadDetails = &models.Thread{}
			storage.db.QueryRow(queryThread, postDetails.PostDetails.Thread).
				Scan(&postDetails.ThreadDetails.ID, &postDetails.ThreadDetails.Slug,
					&postDetails.ThreadDetails.Title, &postDetails.ThreadDetails.Message,
					&postDetails.ThreadDetails.Forum, &postDetails.ThreadDetails.Author,
					&postDetails.ThreadDetails.Created, &postDetails.ThreadDetails.Votes)
		}
	}
	return &postDetails, http.StatusOK
}

func (storage *Storage) UpdatePostDetails(id *string, postUpd *models.PostUpdate) (*models.Post, int) {
	tx, err := storage.db.Begin()
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	defer func(tx *pgx.Tx) {
		_ = tx.Commit()
	}(tx)

	postUpdated := models.Post{}

	err = tx.QueryRow("UPDATE posts SET message=coalesce($2,message), is_edited=(CASE WHEN $2 IS NULL OR $2 = message THEN FALSE ELSE TRUE END) WHERE ID=$1 RETURNING id, author::TEXT, message, created_at, forum::TEXT, thread, is_edited, parent", id, postUpd.Message).
		Scan(&postUpdated.ID, &postUpdated.Author, &postUpdated.Message,
			&postUpdated.Created, &postUpdated.Forum, &postUpdated.Thread,
			&postUpdated.IsEdited, &postUpdated.Parent)
	if err != nil {
		return nil, http.StatusNotFound
	}

	return &postUpdated, http.StatusOK
}
