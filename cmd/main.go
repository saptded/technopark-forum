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

	// user
	router.POST("/api/user/:nickname/create", api.CreateUser)

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

	log.Println("server start on 8080 port")
	err = fasthttp.ListenAndServe(":8080", router.Handler)
	if err != nil {
		log.Fatalf("server failed: %s", err.Error())
	}
}
