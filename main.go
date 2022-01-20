package main

import (
	"github.com/buaazp/fasthttprouter"
	"github.com/jackc/pgx"
	"github.com/valyala/fasthttp"
	"log"
	"technopark-forum/delivery"
	"technopark-forum/repository"
	"technopark-forum/usecase"
)

var defaultDBConfig = pgx.ConnPoolConfig{
	ConnConfig: pgx.ConnConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Database: "docker",
		User:     "docker",
		Password: "docker",
	},
	MaxConnections: 100,
}

func initDB() (*pgx.ConnPool, error) {
	db, err := pgx.NewConnPool(defaultDBConfig)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initRouter(api *delivery.Api) *fasthttprouter.Router {
	router := fasthttprouter.New()

	// service
	router.GET("/api/service/status", api.GetStatus)
	router.GET("/api/service/clear", api.Clear)

	// user
	router.POST("/api/user/:nickname/create", api.CreateUser)
	router.GET("/api/user/:nickname/profile", api.GetUserProfile)
	router.POST("/api/user/:nickname/profile", api.UpdateUserProfile)

	// forum
	router.POST("/api/forum", api.CreateForum)
	router.GET("/api/forum/:slug/details", api.GetForum)
	router.POST("/api/forum/:slug/create", api.CreateThread)
	router.GET("/api/forum/:slug/users", api.GetUsers)
	router.GET("/api/forum/:slug/threads", api.GetThreads)

	// thread
	router.POST("/api/thread/:slug_or_id/create", api.CreatePosts)
	router.GET("/api/thread/:slug_or_id/details", api.GetThread)
	router.POST("/api/thread/:slug_or_id/details", api.UpdateThread)
	router.GET("/api/thread/:slug_or_id/posts", api.GetPosts)
	router.POST("/api/thread/:slug_or_id/vote", api.Vote)

	// post
	router.GET("/api/post/:id/details", api.GetPostDetails)
	router.POST("/api/post/:id/details", api.UpdatePost)

	return router
}

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatalf("initDB failed: %s", err.Error())
	}
	defer db.Close()

	repo := repository.NewForumStorage(db)
	service := usecase.NewForumService(repo)
	api := delivery.NewApi(service)

	router := initRouter(api)

	log.Println("server start on 5000 port")
	err = fasthttp.ListenAndServe(":5000", router.Handler)
	if err != nil {
		log.Fatalf("server failed: %s", err.Error())
	}
}
